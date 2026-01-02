load("@rules_cc//cc:defs.bzl", "cc_library")

# nlohmann/json is header-only, so we can use a simple cc_library
cc_library(
    name = "json",
    hdrs = glob(["include/**/*.hpp"]),
    includes = ["include"],
    visibility = ["//visibility:public"],
)
