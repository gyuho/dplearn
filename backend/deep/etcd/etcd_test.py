#!/usr/bin/env python
"""
This script tests etcd clients.
"""

# etcd client implementation
import etcd

# TODO(gyuho): integration tests, follow Python-way of testing

etcd.put("http://localhost:2379", "foo", "bar")
print etcd.get("http://localhost:2379", "foo")
