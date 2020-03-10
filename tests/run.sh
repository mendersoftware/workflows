#!/bin/sh

py.test -s --tb=short -vv /testing/tests "$@"
