package(default_visibility = ["//visibility:public"])

load("@bazel_tools//tools/build_defs/docker:docker.bzl", "docker_build")

docker_build(
    name = "busybox",
    debs = [
        "@busybox_deb//file",
    ],
    symlinks = {
        "/bin/sh": "/bin/busybox",
        "/usr/bin/busybox": "/bin/busybox",
        "/usr/sbin/busybox": "/bin/busybox",
        "/sbin/busybox": "/bin/busybox",
    },
)

docker_build(
    name = "busybox-libc",
    base = ":busybox",
    debs = [
        "@libc_deb//file",
    ],
)

# TODO: Probably need kmod also
docker_build(
    name = "krouton-network-agent",
    base = ":busybox-libc",
    cmd = ["/krouton-network-agent"],
    files = [
        "//cmd/krouton-network-agent",
    ],
    repository = "kopeio",
)
