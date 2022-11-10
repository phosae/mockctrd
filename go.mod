module github.com/phosae/mockctrd

go 1.18

require (
	github.com/containerd/go-cni v1.1.7
	k8s.io/apimachinery v0.23.1
	k8s.io/cri-api v0.23.1
)

require (
	github.com/containernetworking/cni v1.1.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20211209124913-491a49abca63 // indirect
	golang.org/x/sys v0.0.0-20210831042530-f4d43177bf5e // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.27.1 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace (
	github.com/containerd/go-cni => ./go-cni
	github.com/containernetworking/cni => ./cni
)
