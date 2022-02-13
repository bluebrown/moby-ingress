package container

type Service struct {
	Name         string
	Labels       map[string]string
	Replicas     uint64
	EndpointMode string
}
