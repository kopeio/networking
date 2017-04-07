package(default_visibility = ["//visibility:public"])

licenses(["notice"])

load(
    "@io_bazel_rules_go//go:def.bzl",
    "go_library",
)

go_library(
    name = "go_default_library",
    srcs = [
        "helpers.go",
        "link_addresses.go",
        "links.go",
        "neigh_table.go",
        "route_table.go",
        "xfrm_policy_table.go",
        "xfrm_state_table.go",
    ],
    deps = [
        "//pkg/util:go_default_library",
        "@com_github_golang_glog//:go_default_library",
        "@com_github_vishvananda_netlink//:go_default_library",
    ],
)
