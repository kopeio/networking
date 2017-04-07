package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
)

go_library(
    name = "go_default_library",
    srcs = [
        "authentication_strategy.go",
        "encapsulation_strategy.go",
        "encryption_strategy.go",
        "esp_listener.go",
        "ipsecrouting.go",
    ],
    deps = [
        "//pkg/routing:go_default_library",
        "//pkg/routing/netutil:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_vishvananda_netlink//:go_default_library",
    ],
)
