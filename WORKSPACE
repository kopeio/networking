git_repository(
    name = "io_bazel_rules_go",
    remote = "https://github.com/bazelbuild/rules_go.git",
    tag = "0.3.0",
)

load("@io_bazel_rules_go//go:def.bzl", "go_repositories", "new_go_repository")

go_repositories()

# for building docker base images
debs = (
    (
        "busybox_deb",
        "51651980a993b02c8dc663a5539a4d83704e56c2fed93dd8d1b2580e61319af5",
        "http://ftp.us.debian.org/debian/pool/main/b/busybox/busybox-static_1.22.0-19_amd64.deb",
    ),
    (
        "libc_deb",
        "ee4d9dea08728e2c2bbf43d819c3c7e61798245fab4b983ae910865980f791ad",
        "http://ftp.us.debian.org/debian/pool/main/g/glibc/libc6_2.19-18+deb8u6_amd64.deb",
    )
)

[http_file(
    name = name,
    sha256 = sha256,
    url = url,
) for name, sha256, url in debs]


new_go_repository(
    name = "com_github_golang_glog",
    importpath = "github.com/golang/glog",
    commit = "23def4e6c14b4da8ac2ed8007337bc5eb5007998",
)

new_go_repository(
    name = "com_github_vishvananda_netlink",
    importpath = "github.com/vishvananda/netlink",
    commit = "ffec63e1f1d04e356402671aff3f8da58b5dc585",
)

new_go_repository(
    name = "com_github_vishvananda_netns",
    importpath = "github.com/vishvananda/netns",
    commit = "8ba1072b58e0c2a240eb5f6120165c7776c3e7b8",
)




## CLIENT-GO DEPS
new_go_repository(
    name = "io_k8s_client_go",
    importpath = "k8s.io/client-go",
    commit = "b22087a53becae45931ed72d5e0f12e0031d771a",
)


new_go_repository(
    name = "com_github_gogo_protobuf",
    importpath = "github.com/gogo/protobuf",
    commit = "a9cd0c35b97daf74d0ebf3514c5254814b2703b4",
)

new_go_repository(
    name = "com_github_ugorji_go",
    importpath = "github.com/ugorji/go",
    commit = "faddd6128c66c4708f45fdc007f575f75e592a3c",
)

new_go_repository(
    name = "com_github_docker_distribution",
    importpath = "github.com/docker/distribution",
    commit = "93a48e361cbe06d1e77e9d17d02fa6db192a8ca5",
)

new_go_repository(
    name = "com_github_davecgh_go_spew",
    importpath = "github.com/davecgh/go-spew",
    commit = "346938d642f2ec3594ed81d874461961cd0faa76",
)

new_go_repository(
    name = "com_github_google_gofuzz",
    importpath = "github.com/google/gofuzz",
    commit = "fd52762d25a41827db7ef64c43756fd4b9f7e382",
)

new_go_repository(
    name = "com_github_go_openapi_spec",
    importpath = "github.com/go-openapi/spec",
    commit = "8f2b3d0e3aa15100eea0ab61dc6fa02f00f5e713",
)

new_go_repository(
    name = "in_gopkg_inf_v0",
    importpath = "gopkg.in/inf.v0",
    commit = "3887ee99ecf07df5b447e9b00d9c0b2adaa9f3e4",
)

new_go_repository(
    name = "com_github_emicklei_go_restful",
    importpath = "github.com/emicklei/go-restful",
    commit = "3d66f886316ac990eb502aaa89ea38546420b8b7",
)

new_go_repository(
    name = "com_github_spf13_pflag",
    importpath = "github.com/spf13/pflag",
    commit = "5ccb023bc27df288a957c5e994cd44fd19619465",
)

new_go_repository(
    name = "org_golang_x_net",
    importpath = "golang.org/x/net",
    commit = "fca93ed71fff8fa71efeaf6501adc8921a6da7bc",
)

new_go_repository(
    name = "com_github_go_openapi_jsonreference",
    importpath = "github.com/go-openapi/jsonreference",
    commit = "13c6e3589ad90f49bd3e3bbe2c2cb3d7a4142272",
)

new_go_repository(
    name = "com_github_go_openapi_swag",
    importpath = "github.com/go-openapi/swag",
    commit = "3b6d86cd965820f968760d5d419cb4add096bdd7",
)

new_go_repository(
    name = "com_github_go_openapi_jsonpointer",
    importpath = "github.com/go-openapi/jsonpointer",
    commit = "46af16f9f7b149af66e5d1bd010e3574dc06de98",
)

new_go_repository(
    name = "com_github_pborman_uuid",
    importpath = "github.com/pborman/uuid",
    commit = "3d4f2ba23642d3cfd06bd4b54cf03d99d95c0f1b",
)

new_go_repository(
    name = "com_github_mailru_easyjson",
    importpath = "github.com/mailru/easyjson",
    commit = "06715e4cffead4add5840414232aaf7786809ea9",
)

new_go_repository(
    name = "com_github_PuerkitoBio_purell",
    importpath = "github.com/PuerkitoBio/purell",
    commit = "8a290539e2e8629dbc4e6bad948158f790ec31f4",
)

new_go_repository(
    name = "com_github_PuerkitoBio_urlesc",
    importpath = "github.com/PuerkitoBio/urlesc",
    commit = "5bd2802263f21d8788851d5305584c82a5c75d7e",
)

new_go_repository(
    name = "org_golang_x_text",
    importpath = "golang.org/x/text",
    commit = "a8b38433e35b65ba247bb267317037dee1b70cea",
)

new_go_repository(
    name = "com_github_ghodss_yaml",
    importpath = "github.com/ghodss/yaml",
    commit = "bea76d6a4713e18b7f5321a2b020738552def3ea",
)

new_go_repository(
    name = "in_gopkg_yaml_v2",
    importpath = "gopkg.in/yaml.v2",
    commit = "a5b47d31c556af34a302ce5d659e6fea44d90de0",
)

new_go_repository(
    name = "com_github_juju_ratelimit",
    importpath = "github.com/juju/ratelimit",
    commit = "77ed1c8a01217656d2080ad51981f6e99adaa177",
)

new_go_repository(
    name = "com_github_coreos_go_oidc",
    importpath = "github.com/coreos/go-oidc",
    commit = "16c5ecc505f1efa0fe4685826fd9962c4d137e87",
)

new_go_repository(
    name = "com_github_blang_semver",
    importpath = "github.com/blang/semver",
    commit = "60ec3488bfea7cca02b021d106d9911120d25fe9",
)

new_go_repository(
    name = "org_golang_x_oauth2",
    importpath = "golang.org/x/oauth2",
    commit = "25b4fb1468cb89700c7c060cb99f30581a61f5e3",
)

new_go_repository(
    name = "com_github_jonboulle_clockwork",
    importpath = "github.com/jonboulle/clockwork",
    commit = "bcac9884e7502bb2b474c0339d889cb981a2f27f",
)

new_go_repository(
    name = "com_github_coreos_pkg",
    importpath = "github.com/coreos/pkg",
    commit = "447b7ec906e523386d9c53be15b55a8ae86ea944",
)

new_go_repository(
    name = "com_google_cloud_go",
    importpath = "cloud.google.com/go",
    commit = "9d7cefd9e5d1576fda4831915bce6677af4b60ef",
)
