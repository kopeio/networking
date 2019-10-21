module kope.io/networking

// Lock some versions to kubernetes 1.16.2, as required by controller-runtime
// replace X => X kubernetes-1.16.1
replace k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a

replace k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8

replace k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48

require (
	github.com/ghodss/yaml v1.0.0
	github.com/googleapis/gnostic v0.1.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20170926212834-c1f8028e62ad // indirect
	github.com/vishvananda/netlink v1.0.0
	github.com/vishvananda/netns v0.0.0-20171111001504-be1fbeda1936 // indirect
	k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/client-go v2.0.0-alpha.0.0.20181026185218-bf181536cb4d+incompatible
	k8s.io/klog v1.0.0
)
