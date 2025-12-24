# Removed Features

This document tracks features that were intentionally removed and are not currently supported.

## CLI Mode
**What it did**: Printed file coverage report to console (files not in any Bazel target)

**Why removed**: Focus shifted to web UI for better visualization and interactivity

**Restoration**: Can be restored from git history if needed for automation/CI use cases

## File Coverage Analysis
**What it did**: Found all source files and compared to Bazel targets to find uncovered files

**Why removed**: Not part of core dependency analysis mission

**Restoration**: Could be added back as optional CLI/web feature if needed
