package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_binary",
    "go_library",
)

go_binary(
    name = "networking-agent",
    gc_linkopts = ["-s"],
    library = ":go_default_library",
)

go_library(
    name = "go_default_library",
    srcs = [
        "main.go",
        "options.go",
    ],
    visibility = ["//visibility:private"],
    deps = [
        "@com_github_ghodss_yaml//:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_spf13_pflag//:go_default_library",
        "@io_k8s_client_go//kubernetes:go_default_library",
        "@io_k8s_client_go//pkg/api/v1:go_default_library",
        "@io_k8s_client_go//rest:go_default_library",
        "//pkg/routing:go_default_library",
        "//pkg/routing/ipsec:go_default_library",
        "//pkg/routing/layer2:go_default_library",
        "//pkg/routing/vxlan:go_default_library",
        "//pkg/watchers:go_default_library",
    ],
)
