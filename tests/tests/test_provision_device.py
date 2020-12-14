import requests
import time
import json

def test_provision_device(mmock_url, workflows_url):
    # start the provision_device workflow
    res = requests.post(
        workflows_url + "/api/v1/workflow/provision_device",
        json={
            "request_id": "1234567890",
            "authorization": "Bearer TEST",
            "device_id": "1",
            "tenant_id": "123456789012345678901234",
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
    assert response["status"] == "done"
    assert {"name": "request_id", "value": "1234567890"} in response["inputParameters"]
    assert {"name": "authorization", "value": "Bearer TEST"} in response[
        "inputParameters"
    ]
    assert {"name": "device_id", "value": "1"} in response["inputParameters"]
    assert {"name": "tenant_id", "value": "123456789012345678901234"} in response[
        "inputParameters"
    ]
    assert len(response["results"]) == 2
    assert response["results"][0]["success"] == True
    assert response["results"][0]["httpResponse"]["statusCode"] == 200
    assert response["results"][1]["success"] == True
    assert response["results"][1]["httpResponse"]["statusCode"] == 201
    #  verify the mock server has been correctly called
    res = requests.get(mmock_url + "/api/request/all")
    assert res.status_code == 200
    response = res.json()
    expected = [
        {
            "request": {
                "scheme": "http",
                "host": "mender-inventory",
                "port": "8080",
                "method": "POST",
                "path": "/api/internal/v1/inventory/devices",
                "queryStringParameters": {},
                "fragment": "",
                "headers": {
                    "Accept-Encoding": ["gzip"],
                    "Authorization": ["Bearer TEST"],
                    "Content-Length": ["10"],
                    "Content-Type": ["application/json"],
                    "User-Agent": ["Go-http-client/1.1"],
                    "X-Men-Requestid": ["1234567890"],
                },
                "cookies": {},
                "body": '{"id":"1"}',
            },
        },
        {
            "request": {
                "scheme": "http",
                "host": "mender-deviceconnect",
                "port": "8080",
                "method": "POST",
                "path": "/api/internal/v1/deviceconnect/tenants/"
                + "123456789012345678901234/devices",
                "queryStringParameters": {},
                "fragment": "",
                "headers": {
                    "Accept-Encoding": ["gzip"],
                    "Content-Length": ["17"],
                    "Content-Type": ["application/json"],
                    "User-Agent": ["Go-http-client/1.1"],
                    "X-Men-Requestid": ["1234567890"],
                },
                "cookies": {},
                "body": '{"device_id":"1"}',
            },
        },
    ]
    assert expected[0]["request"] == response[0]["request"]
    assert expected[1]["request"] == response[1]["request"]
