package coordinator

type Service struct {
	cfg Config
	db  DataStorage
}

func NewService(cfg Config, db DataStorage) *Service {
	return &Service{cfg: cfg, db: db}
}
