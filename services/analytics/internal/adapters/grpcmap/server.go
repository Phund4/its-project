// Package grpcmap — gRPC API данных для map_portal.
package grpcmap

import (
	"context"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	mapv1 "traffic-analytics/api/map/v1"
	"traffic-analytics/internal/adapters/clickhouse"
	"traffic-analytics/internal/config"
	"traffic-analytics/internal/portalhub"
)

// Server реализует map.v1.MapPortal.
type Server struct {
	// UnimplementedMapPortalServer заглушки для совместимости gRPC.
	mapv1.UnimplementedMapPortalServer

	// Store чтение ClickHouse (справочники и OLAP).
	Store *clickhouse.Store

	// Hub in-memory автобусы для карты.
	Hub *portalhub.Hub

	// Cfg настройки (INFRA_SIM_DATABASE, …).
	Cfg config.Config
}

// Register регистрирует сервис на gRPC-сервере.
func Register(g *grpc.Server, store *clickhouse.Store, hub *portalhub.Hub, cfg config.Config) {
	mapv1.RegisterMapPortalServer(g, &Server{Store: store, Hub: hub, Cfg: cfg})
}

// ListMunicipalities возвращает города из ClickHouse.
func (s *Server) ListMunicipalities(ctx context.Context, _ *mapv1.ListMunicipalitiesRequest) (*mapv1.ListMunicipalitiesResponse, error) {
	rows, err := s.Store.ListInfraMunicipalities(ctx, s.Cfg.InfraSimDatabase)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "clickhouse: %v", err)
	}
	out := &mapv1.ListMunicipalitiesResponse{Items: make([]*mapv1.Municipality, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, &mapv1.Municipality{
			MunicipalityId: r.MunicipalityID,
			NameRu:         r.NameRU,
			CenterLat:      r.CenterLat,
			CenterLon:      r.CenterLon,
			DefaultZoom:    uint32(r.DefaultZoom),
		})
	}
	return out, nil
}

// ListStops — остановки; продлевает активность города для приёма телеметрии.
func (s *Server) ListStops(ctx context.Context, req *mapv1.ListStopsRequest) (*mapv1.ListStopsResponse, error) {
	mid := strings.TrimSpace(req.GetMunicipalityId())
	if mid == "" {
		return nil, status.Error(codes.InvalidArgument, "municipality_id required")
	}
	s.Hub.TouchMunicipality(mid)
	rows, err := s.Store.ListInfraStops(ctx, s.Cfg.InfraSimDatabase, mid)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "clickhouse: %v", err)
	}
	out := &mapv1.ListStopsResponse{Items: make([]*mapv1.Stop, 0, len(rows))}
	for _, r := range rows {
		out.Items = append(out.Items, &mapv1.Stop{
			StopId:    r.StopID,
			StopCode:  r.StopCode,
			Name:      r.Name,
			NameShort: r.NameShort,
			Lat:       r.Lat,
			Lon:       r.Lon,
		})
	}
	return out, nil
}

// ListBuses — последние позиции из памяти analytics.
func (s *Server) ListBuses(_ context.Context, req *mapv1.ListBusesRequest) (*mapv1.ListBusesResponse, error) {
	mid := strings.TrimSpace(req.GetMunicipalityId())
	if mid == "" {
		return nil, status.Error(codes.InvalidArgument, "municipality_id required")
	}
	s.Hub.TouchMunicipality(mid)
	snaps := s.Hub.ListBuses(mid)
	out := &mapv1.ListBusesResponse{Items: make([]*mapv1.Bus, 0, len(snaps))}
	for _, b := range snaps {
		out.Items = append(out.Items, &mapv1.Bus{
			VehicleId:         b.VehicleID,
			RouteId:           b.RouteID,
			Lat:               b.Lat,
			Lon:               b.Lon,
			SpeedKmh:          b.SpeedKmh,
			HeadingDeg:        b.HeadingDeg,
			ObservedAtRfc3339: b.ObservedAtRfc3339,
		})
	}
	return out, nil
}
