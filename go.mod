module github.com/phosae/mockctrd

go 1.18

require (
	github.com/containerd/go-cni v1.1.7
	k8s.io/apimachinery v0.25.4
	k8s.io/cri-api v0.23.1
)

require (
	github.com/containernetworking/cni v1.1.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	golang.org/x/net v0.0.0-20220722155237-a158d28d115b // indirect
	golang.org/x/sys v0.0.0-20220520151302-bc2c85ada10a // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/genproto v0.0.0-20210831024726-fe130286e0e2 // indirect
	google.golang.org/grpc v1.40.0 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
)

replace (
	github.com/containerd/go-cni => ./go-cni
	github.com/containernetworking/cni => ./cni
)
