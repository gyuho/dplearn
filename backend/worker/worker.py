"""
This script interacts with backend/web/queue-service.
"""

from __future__ import print_function

import base64
import json
import sys
import time
import requests

def fetch(bucket):
    """
    fetch fetches a scheduled job from queue service.
    """
    return ''

def post(bucket, item):
    """
    post posts the processed job to the queue service.
    """
    return ''


"""

func testWorker(ep string, item *etcdqueue.Item) {
	orig := *item
	time.Sleep(10 * time.Second)

	glog.Infof("[TEST] fetching from %q", ep)
	resp, err := http.Get(ep)
	if err != nil {
		panic(err)
	}

	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	io.Copy(ioutil.Discard, resp.Body)
	resp.Body.Close()

	var front etcdqueue.Item
	if err = json.Unmarshal(rb, &front); err != nil {
		panic(err)
	}
	if !reflect.DeepEqual(front, orig) {
		glog.Warningf("front expected %+v, got %+v", front, orig)
	}

	time.Sleep(10 * time.Second)

	orig.Progress = 100
	orig.Value = "new-value"
	bts, err := json.Marshal(orig)
	if err != nil {
		panic(err)
	}
	if _, err := http.Post(ep, "application/json", bytes.NewReader(bts)); err != nil {
		panic(err)
	}
	glog.Infof("[TEST] posted to %q", ep)
}

"""