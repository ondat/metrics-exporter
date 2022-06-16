module github.com/ondat/metrics-exporter

go 1.17

require (
	github.com/google/uuid v1.3.0
	github.com/prometheus/client_golang v1.12.2
	github.com/prometheus/procfs v0.7.3
	github.com/stretchr/testify v1.7.1
	go.uber.org/zap v1.21.0
	golang.org/x/sys v0.0.0-20220517195934-5e4e11fc645e
	k8s.io/apimachinery v0.21.13
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/go-logr/logr => github.com/go-logr/logr v0.4.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.21.13
	k8s.io/controller-runtime => k8s.io/controller-runtime v0.8.3
	k8s.io/klog/v2 => k8s.io/klog/v2 v2.4.0
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.8.3
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.1.2 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/go-logr/logr v1.2.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/google/go-cmp v0.5.5 // indirect
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.2-0.20181231171920-c182affec369 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/prometheus/client_model v0.2.0 // indirect
	github.com/prometheus/common v0.34.0 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/multierr v1.8.0 // indirect
	golang.org/x/net v0.0.0-20220225172249-27dd8689420f // indirect
	golang.org/x/text v0.3.7 // indirect
	google.golang.org/protobuf v1.28.0 // indirect
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20220512140231-539c8e751b99 // indirect
	k8s.io/klog/v2 v2.9.0 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.2.1 // indirect
	sigs.k8s.io/yaml v1.3.0 // indirect
)
