// Command bus-telemetry-generator — шлёт BusTelemetry по gRPC: несколько автобусов на город,
// позиции вдоль реальных трасс (совпадают с its_infra_sim из 002_seed.sql).
// Режим BUS_TELEMETRY_ALL_CITIES=true шлёт телеметрию по всем городам — в analytics без TouchMunicipality
// для неоткрытого города данные отбрасываются (portalhub TTL).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	busv1 "data-ingestion/api/bus/v1"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type latLon struct {
	lat float64
	lon float64
}

// routeDef маршрут симуляции; координаты синхронизированы с infra/clickhouse/002_seed.sql.
type routeDef struct {
	municipalityID string
	routeNum       string
	segmentID      string
	loop           bool
	poly           []latLon
}

// Координаты — приближение к OSM (Ленинградский, Невский, Казань центр, пр. Ленина Екб).
var simRoutes = []routeDef{
	{
		municipalityID: "msk",
		routeNum:       "42",
		segmentID:      "leningradsky-r42",
		loop:           true,
		poly: []latLon{
			{55.77785, 37.58465},
			{55.78255, 37.57845},
			{55.78775, 37.57125},
			{55.79335, 37.56395},
			{55.79905, 37.55655},
			{55.80485, 37.54915},
			{55.81065, 37.54175},
			{55.81695, 37.53395},
		},
	},
	{
		municipalityID: "spb",
		routeNum:       "15",
		segmentID:      "nevsky-r15",
		loop:           true,
		poly: []latLon{
			{59.94035, 30.31455},
			{59.93975, 30.32205},
			{59.93885, 30.32955},
			{59.93755, 30.33705},
			{59.93625, 30.34455},
			{59.93495, 30.35205},
			{59.93355, 30.35955},
			{59.93185, 30.36805},
		},
	},
	{
		municipalityID: "kzn",
		routeNum:       "7",
		segmentID:      "kazan-center-r7",
		loop:           true,
		poly: []latLon{
			{55.79965, 49.09745},
			{55.79855, 49.10385},
			{55.79745, 49.11025},
			{55.79635, 49.11665},
			{55.79525, 49.12305},
			{55.79415, 49.12945},
			{55.79305, 49.13585},
		},
	},
	{
		municipalityID: "ekb",
		routeNum:       "22",
		segmentID:      "lenina-r22",
		loop:           true,
		poly: []latLon{
			{56.84185, 60.59685},
			{56.84105, 60.60235},
			{56.84015, 60.60785},
			{56.83925, 60.61335},
			{56.83835, 60.61885},
			{56.83745, 60.62435},
			{56.83655, 60.62985},
			{56.83565, 60.63535},
		},
	},
}

func routesByMunicipality() map[string]routeDef {
	m := make(map[string]routeDef, len(simRoutes))
	for _, r := range simRoutes {
		m[r.municipalityID] = r
	}
	return m
}

const earthRadiusM = 6371000.0

func toRad(d float64) float64 { return d * math.Pi / 180 }

func haversineM(a, b latLon) float64 {
	dLat := toRad(b.lat - a.lat)
	dLon := toRad(b.lon - a.lon)
	s1 := math.Sin(dLat / 2)
	s2 := math.Sin(dLon / 2)
	x := s1*s1 + math.Cos(toRad(a.lat))*math.Cos(toRad(b.lat))*s2*s2
	return 2 * earthRadiusM * math.Asin(math.Min(1, math.Sqrt(x)))
}

func bearingDeg(from, to latLon) float64 {
	dLon := toRad(to.lon - from.lon)
	y := math.Sin(dLon) * math.Cos(toRad(to.lat))
	x := math.Cos(toRad(from.lat))*math.Sin(toRad(to.lat)) - math.Sin(toRad(from.lat))*math.Cos(toRad(to.lat))*math.Cos(dLon)
	brng := math.Atan2(y, x)
	return math.Mod(math.Mod(brng*180/math.Pi, 360)+360, 360)
}

func polylineLengthM(poly []latLon, loop bool) float64 {
	if len(poly) < 2 {
		return 0
	}
	var sum float64
	for i := 0; i < len(poly)-1; i++ {
		sum += haversineM(poly[i], poly[i+1])
	}
	if loop {
		sum += haversineM(poly[len(poly)-1], poly[0])
	}
	return sum
}

// posOnRoute возвращает lat, lon, курс (градусы) на дистанции distM от начала (по замкнутому контуру при loop).
func posOnRoute(poly []latLon, loop bool, distM float64) (float64, float64, float64) {
	if len(poly) == 0 {
		return 0, 0, 0
	}
	if len(poly) == 1 {
		return poly[0].lat, poly[0].lon, 0
	}
	total := polylineLengthM(poly, loop)
	if total <= 0 {
		return poly[0].lat, poly[0].lon, 0
	}
	d := math.Mod(distM, total)
	var segStart latLon
	var segEnd latLon
	acc := 0.0
	nSeg := len(poly) - 1
	if loop {
		nSeg = len(poly)
	}
	for seg := 0; seg < nSeg; seg++ {
		segStart = poly[seg]
		if seg == len(poly)-1 {
			segEnd = poly[0]
		} else {
			segEnd = poly[seg+1]
		}
		segLen := haversineM(segStart, segEnd)
		if d <= acc+segLen || seg == nSeg-1 {
			t := 0.0
			if segLen > 1e-6 {
				t = (d - acc) / segLen
			}
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			lat := segStart.lat + t*(segEnd.lat-segStart.lat)
			lon := segStart.lon + t*(segEnd.lon-segStart.lon)
			hdg := bearingDeg(segStart, segEnd)
			return lat, lon, hdg
		}
		acc += segLen
	}
	return poly[0].lat, poly[0].lon, bearingDeg(poly[0], poly[1])
}

type busState struct {
	route   routeDef
	vehicle string
	alongM  float64
}

func envBool(key string, def bool) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	if v == "" {
		return def
	}
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return def
	}
}

func envInt(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return def
	}
	return n
}

func envFloat(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 {
		return def
	}
	return f
}

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))

	if err := tryLoadEnvFile(); err != nil {
		slog.Error("load env file", "err", err)
		os.Exit(1)
	}

	addr := strings.TrimSpace(os.Getenv("BUS_TELEMETRY_GRPC_ADDR"))
	if addr == "" {
		addr = "127.0.0.1:50051"
	}
	muniFilter := strings.TrimSpace(os.Getenv("BUS_TELEMETRY_MUNICIPALITY_ID"))
	allCities := envBool("BUS_TELEMETRY_ALL_CITIES", true)

	interval := 5 * time.Second
	if v := strings.TrimSpace(os.Getenv("BUS_TELEMETRY_INTERVAL_SEC")); v != "" {
		if sec, err := time.ParseDuration(v + "s"); err == nil && sec > 0 {
			interval = sec
		}
	}

	speedKmh := envFloat("BUS_TELEMETRY_SPEED_KMH", 52)
	busesPerCity := envInt("BUS_TELEMETRY_BUSES_PER_CITY", 4)
	speedMs := speedKmh / 3.6
	stepM := speedMs * interval.Seconds()

	byMuni := routesByMunicipality()
	var active []routeDef
	if allCities {
		active = append(active, simRoutes...)
	} else {
		if muniFilter == "" {
			muniFilter = "msk"
		}
		r, ok := byMuni[muniFilter]
		if !ok {
			slog.Warn("unknown BUS_TELEMETRY_MUNICIPALITY_ID, fallback msk", "id", muniFilter)
			r = byMuni["msk"]
			muniFilter = "msk"
		}
		active = append(active, r)
	}

	var buses []busState
	for _, rd := range active {
		total := polylineLengthM(rd.poly, rd.loop)
		for i := 0; i < busesPerCity; i++ {
			var start float64
			if busesPerCity > 0 && total > 0 {
				start = (total / float64(busesPerCity)) * float64(i)
			}
			buses = append(buses, busState{
				route:   rd,
				vehicle: fmt.Sprintf("%s-bus-%02d", rd.municipalityID, i+1),
				alongM:  start,
			})
		}
	}

	rootCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		slog.Error("grpc dial", "addr", addr, "err", err)
		os.Exit(1)
	}
	defer conn.Close()

	cli := busv1.NewBusTelemetryServiceClient(conn)
	tick := time.NewTicker(interval)
	defer tick.Stop()

	slog.Info("bus telemetry generator",
		"grpc", addr,
		"interval", interval,
		"speed_kmh", speedKmh,
		"buses_per_city", busesPerCity,
		"vehicles", len(buses),
		"all_cities", allCities,
		"single_municipality_id", muniFilter,
	)

	var tickN int
	for {
		tickN++
		ts := time.Now().UTC().Format(time.RFC3339Nano)
		var okN, errN int

		for i := range buses {
			b := &buses[i]
			lat, lon, hdg := posOnRoute(b.route.poly, b.route.loop, b.alongM)
			b.alongM += stepM

			_, err := cli.SendBusTelemetry(rootCtx, &busv1.BusTelemetry{
				SegmentId:         b.route.segmentID,
				VehicleId:         b.vehicle,
				RouteId:           b.route.routeNum,
				Lat:               lat,
				Lon:               lon,
				SpeedKmh:          speedKmh,
				HeadingDeg:        hdg,
				ObservedAtRfc3339: ts,
				MunicipalityId:    b.route.municipalityID,
			})
			if err != nil {
				errN++
				slog.Error("SendBusTelemetry", "vehicle", b.vehicle, "err", err)
			} else {
				okN++
			}
		}

		if tickN == 1 || tickN%15 == 0 {
			slog.Info("telemetry tick", "tick", tickN, "ok", okN, "errors", errN,
				"hint", "откройте город на карте — иначе hub отбросит телеметрию до TouchMunicipality")
		}

		select {
		case <-rootCtx.Done():
			return
		case <-tick.C:
		}
	}
}
