load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/rules_go/releases/download/v0.20.1/rules_go-v0.20.1.tar.gz",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.20.1/rules_go-v0.20.1.tar.gz",
    ],
    sha256 = "842ec0e6b4fbfdd3de6150b61af92901eeb73681fd4d185746644c338f51d4c0",
)

http_archive(
    name = "bazel_gazelle",
    urls = [
        "https://storage.googleapis.com/bazel-mirror/github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.0/bazel-gazelle-v0.19.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.19.0/bazel-gazelle-v0.19.0.tar.gz",
    ],
    sha256 = "41bff2a0b32b02f20c227d234aa25ef3783998e5453f7eade929704dcff7cd4b",
)

load("@io_bazel_rules_go//go:deps.bzl", "go_rules_dependencies", "go_register_toolchains")

go_rules_dependencies()

go_register_toolchains()

load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies")

gazelle_dependencies()

#=============================================================================
# Docker rules

http_archive(
    name = "io_bazel_rules_docker",
    sha256 = "413bb1ec0895a8d3249a01edf24b82fd06af3c8633c9fb833a0cb1d4b234d46d",
    strip_prefix = "rules_docker-0.12.0",
    urls = ["https://github.com/bazelbuild/rules_docker/releases/download/v0.12.0/rules_docker-v0.12.0.tar.gz"],
)

load(
    "@io_bazel_rules_docker//repositories:repositories.bzl",
    container_repositories = "repositories",
)

container_repositories()

# This is NOT needed when going through the language lang_image
# "repositories" function(s).
load("@io_bazel_rules_docker//repositories:deps.bzl", container_deps = "deps")

container_deps()

load(
    "@io_bazel_rules_docker//container:container.bzl",
    "container_pull",
)

container_pull(
    name = "debian-base-amd64",
    digest = "sha256:5f25d97ece9076612b64bb551e12f1e39a520176b684e2d663ce1bd53c5d0618",
    registry = "k8s.gcr.io",
    repository = "debian-base-amd64",
    tag = "v1.0.0",  # ignored, but kept here for documentation
)

##=============================================================================
## Go dependencies rules
#
### NETLINK
#
#go_repository(
#    name = "com_github_vishvananda_netlink",
#    commit = "fe3b5664d23a11b52ba59bece4ff29c52772a56b",
#    importpath = "github.com/vishvananda/netlink",
#)
#
#go_repository(
#    name = "com_github_vishvananda_netns",
#    commit = "54f0e4339ce73702a0607f49922aaa1e749b418d",
#    importpath = "github.com/vishvananda/netns",
#)
#
## Deps, per https://github.com/kubernetes/client-go/blob/v2.0.0/Godeps/Godeps.json
#
#### CLIENT-GO DEPS
#go_repository(
#    name = "io_k8s_client_go",
#    commit = "afb4606c45bae77c4dc2c15291d4d7d6d792196c",
#    importpath = "k8s.io/client-go",
#)
#
#go_repository(
#    name = "io_k8s_api",
#    commit = "81aa34336d28aadc3a8e8da7dfd9258c5157e5e4",
#    importpath = "k8s.io/api",
#)
#
#go_repository(
#    name = "io_k8s_apimachinery",
#    commit = "3b05bbfa0a45413bfa184edbf9af617e277962fb",
#    importpath = "k8s.io/apimachinery",
#)
#
#### GLOG
##
##go_repository(
##    name = "com_github_golang_glog",
##    commit = "44145f04b68cf362d9c4df2182967c2275eaefed",
##    importpath = "github.com/golang/glog",
##)
##
#go_repository(
#    name = "com_github_spf13_pflag",
#    commit = "5ccb023bc27df288a957c5e994cd44fd19619465",
#    importpath = "github.com/spf13/pflag",
#)
#
#go_repository(
#    name = "com_github_ghodss_yaml",
#    commit = "73d445a93680fa1a78ae23a5839bad48f32ba1ee",
#    importpath = "github.com/ghodss/yaml",
#)
#
##go_repository(
##    name = "com_github_ugorji_go",
##    commit = "f1f1a805ed361a0e078bb537e4ea78cd37dcf065",
##    importpath = "github.com/ugorji/go",
##)
##
##go_repository(
##    name = "com_github_google_gofuzz",
##    commit = "bbcb9da2d746f8bdbd6a936686a0a6067ada0ec5",
##    importpath = "github.com/google/gofuzz",
##)
##
##go_repository(
##    name = "com_github_gogo_protobuf",
##    commit = "e18d7aa8f8c624c915db340349aad4c49b10d173",
##    importpath = "github.com/gogo/protobuf",
##)
##
##go_repository(
##    name = "com_github_go_openapi_spec",
##    commit = "6aced65f8501fe1217321abf0749d354824ba2ff",
##    importpath = "github.com/go-openapi/spec",
##)
##
##go_repository(
##    name = "com_github_go_openapi_swag",
##    commit = "1d0bd113de87027671077d3c71eb3ac5d7dbba72",
##    importpath = "github.com/go-openapi/swag",
##)
##
##go_repository(
##    name = "in_gopkg_inf_v0",
##    commit = "3887ee99ecf07df5b447e9b00d9c0b2adaa9f3e4",
##    importpath = "gopkg.in/inf.v0",
##)
##
##go_repository(
##    name = "com_github_emicklei_go_restful",
##    commit = "89ef8af493ab468a45a42bb0d89a06fccdd2fb22",
##    importpath = "github.com/emicklei/go-restful",
##)
##
##go_repository(
##    name = "org_golang_x_net",
##    commit = "e90d6d0afc4c315a0d87a568ae68577cc15149a0",
##    importpath = "golang.org/x/net",
##)
##
##go_repository(
##    name = "com_github_go_openapi_jsonreference",
##    commit = "13c6e3589ad90f49bd3e3bbe2c2cb3d7a4142272",
##    importpath = "github.com/go-openapi/jsonreference",
##)
##
##go_repository(
##    name = "com_github_go_openapi_jsonpointer",
##    commit = "46af16f9f7b149af66e5d1bd010e3574dc06de98",
##    importpath = "github.com/go-openapi/jsonpointer",
##)
##
##go_repository(
##    name = "com_github_PuerkitoBio_purell",
##    commit = "8a290539e2e8629dbc4e6bad948158f790ec31f4",
##    importpath = "github.com/PuerkitoBio/purell",
##)
##
##go_repository(
##    name = "com_github_mailru_easyjson",
##    commit = "d5b7844b561a7bc640052f1b935f7b800330d7e0",
##    importpath = "github.com/mailru/easyjson",
##)
##
##go_repository(
##    name = "org_golang_x_text",
##    commit = "2910a502d2bf9e43193af9d68ca516529614eed3",
##    importpath = "golang.org/x/text",
##)
##
##go_repository(
##    name = "com_github_PuerkitoBio_urlesc",
##    commit = "5bd2802263f21d8788851d5305584c82a5c75d7e",
##    importpath = "github.com/PuerkitoBio/urlesc",
##)
##
##go_repository(
##    name = "com_github_docker_distribution",
##    commit = "cd27f179f2c10c5d300e6d09025b538c475b0d51",
##    importpath = "github.com/docker/distribution",
##)
##
##go_repository(
##    name = "com_github_davecgh_go_spew",
##    commit = "5215b55f46b2b919f50a1df0eaa5886afe4e3b3d",
##    importpath = "github.com/davecgh/go-spew",
##)
##
##go_repository(
##    name = "com_github_pborman_uuid",
##    commit = "ca53cad383cad2479bbba7f7a1a05797ec1386e4",
##    importpath = "github.com/pborman/uuid",
##)
#
#go_repository(
#    name = "in_gopkg_yaml_v2",
#    commit = "53feefa2559fb8dfa8d81baad31be332c97d6c77",
#    importpath = "gopkg.in/yaml.v2",
#)
#
##go_repository(
##    name = "com_github_blang_semver",
##    commit = "31b736133b98f26d5e078ec9eb591666edfd091f",
##    importpath = "github.com/blang/semver",
##)
##
##go_repository(
##    name = "com_github_juju_ratelimit",
##    commit = "77ed1c8a01217656d2080ad51981f6e99adaa177",
##    importpath = "github.com/juju/ratelimit",
##)
##
##go_repository(
##    name = "org_golang_x_oauth2",
##    commit = "3c3a985cb79f52a3190fbc056984415ca6763d01",
##    importpath = "golang.org/x/oauth2",
##)
##
##go_repository(
##    name = "com_github_coreos_go_oidc",
##    commit = "5644a2f50e2d2d5ba0b474bc5bc55fea1925936d",
##    importpath = "github.com/coreos/go-oidc",
##)
##
##go_repository(
##    name = "com_github_jonboulle_clockwork",
##    commit = "72f9bd7c4e0c2a40055ab3d0f09654f730cce982",
##    importpath = "github.com/jonboulle/clockwork",
##)
##
##go_repository(
##    name = "com_github_coreos_pkg",
##    commit = "fa29b1d70f0beaddd4c7021607cc3c3be8ce94b8",
##    importpath = "github.com/coreos/pkg",
##)
##
##go_repository(
##    name = "com_google_cloud_go",
##    commit = "3b1ae45394a234c385be014e9a488f2bb6eef821",
##    importpath = "cloud.google.com/go",
##)
#
##go_repository(
##    name = "com_github_ghodss_yaml",
##    commit = "73d445a93680fa1a78ae23a5839bad48f32ba1ee",
##    importpath = "github.com/ghodss/yaml",
##)
##
##go_repository(
##    name = "com_github_spf13_pflag",
##    commit = "9ff6c6923cfffbcd502984b8e0c80539a94968b7",
##    importpath = "github.com/spf13/pflag",
##)
#
#go_repository(
#    name = "com_github_vishvananda_netlink",
#    commit = "177f1ceba557262b3f1c3aba4df93a29199fb4eb",
#    importpath = "github.com/vishvananda/netlink",
#)
