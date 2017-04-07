package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
)

go_library(
    name = "go_default_library",
    srcs = [
        "nodes.go",
        "provider.go",
        "providercontroller.go",
    ],
    deps = [
        "//pkg/util:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@io_k8s_client_go//kubernetes:go_default_library",
        "@io_k8s_client_go//pkg/api/errors:go_default_library",
        "@io_k8s_client_go//pkg/api/unversioned:go_default_library",
        "@io_k8s_client_go//pkg/api/v1:go_default_library",
    ],
)
