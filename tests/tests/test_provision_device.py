# Copyright 2021 Northern.tech AS
#
#    Licensed under the Apache License, Version 2.0 (the "License");
#    you may not use this file except in compliance with the License.
#    You may obtain a copy of the License at
#
#        http://www.apache.org/licenses/LICENSE-2.0
#
#    Unless required by applicable law or agreed to in writing, software
#    distributed under the License is distributed on an "AS IS" BASIS,
#    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#    See the License for the specific language governing permissions and
#    limitations under the License.
import requests
import time


def test_provision_device(mmock_url, workflows_url):
    # start the provision_device workflow
    res = requests.post(
        workflows_url + "/api/v1/workflow/provision_device",
        json={
            "request_id": "1234567890",
            "authorization": "Bearer TEST",
            "device": '{"id": "1"}',
        },
    )
    assert res.status_code == 201
    # verify the response
    response = res.json()
    assert response is not None
    assert type(response) is dict
    assert response["name"] == "provision_device"
    assert response["id"] is not None
    # get the job details, every second until done
    for i in range(10):
        time.sleep(1)
        res = requests.get(
            workflows_url + "/api/v1/workflow/provision_device/" + response["id"]
        )
        assert res.status_code == 200
        # if status is done, break
        response = res.json()
        assert response is not None
        assert type(response) is dict
        if response["status"] == "done":
            break
    # verify the status
    assert {"name": "request_id", "value": "1234567890"} in response["inputParameters"]
    assert {"name": "authorization", "value": "Bearer TEST"} in response[
        "inputParameters"
    ]
    assert {"name": "device", "value": '{"id": "1"}'} in response["inputParameters"]
    assert response["status"] == "done"
    assert len(response["results"]) == 1
    assert response["results"][0]["success"] == True
    assert response["results"][0]["httpResponse"]["statusCode"] == 200
    #  verify the mock server has been correctly called
    res = requests.get(mmock_url + "/api/request/all")
    assert res.status_code == 200
    response = res.json()
    assert len(response) == 1
    expected = {
        "request": {
            "scheme": "http",
            "host": "mender-inventory",
            "port": "8080",
            "method": "POST",
            "path": "/api/internal/v1/inventory/devices",
            "queryStringParameters": {},
            "fragment": "",
            "headers": {
                "Content-Type": ["application/json"],
                "Accept-Encoding": ["gzip"],
                "Authorization": ["Bearer TEST"],
                "Content-Length": ["11"],
                "User-Agent": ["Go-http-client/1.1"],
                "X-Men-Requestid": ["1234567890"],
            },
            "cookies": {},
            "body": '{"id": "1"}',
        },
    }
    assert expected["request"] == response[0]["request"]
