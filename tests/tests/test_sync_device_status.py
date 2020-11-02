import requests
import time


def test_sync_device_status(mmock_url, workflows_url):
    # start the provision_device workflow
    request_id = "1234567890"
    device_status = "accepted"
    device_id = "2"
    tenant_id = "1"
    res = requests.post(
        workflows_url + "/api/v1/workflow/sync_device_status",
        json={
            "request_id": request_id,
            "device_id": device_id,
            "tenant_id": tenant_id,
        },
    )
    assert res.status_code == 201
    # verify the response
    response = res.json()
    assert response is not None
    assert type(response) is dict
    assert response["name"] == "sync_device_status"
    assert response["id"] is not None
    # get the job details, every second until done
    for i in range(10):
        time.sleep(1)
        res = requests.get(
            workflows_url + "/api/v1/workflow/sync_device_status/" + response["id"]
        )
        assert res.status_code == 200
        # if status is done, break
        response = res.json()
        assert response is not None
        assert type(response) is dict
        if response["status"] == "done":
            break
    # verify the status
    assert {"name": "request_id", "value": request_id} in response["inputParameters"]
    assert {"name": "device_id", "value": device_id} in response["inputParameters"]
    assert response["status"] == "done"
    assert len(response["results"]) == 2
    assert response["results"][0]["success"] == True
    assert response["results"][1]["success"] == True
    assert response["results"][0]["httpResponse"]["statusCode"] == 200
    assert response["results"][1]["httpResponse"]["statusCode"] == 200
    #  verify the mock server has been correctly called
    res = requests.get(mmock_url + "/api/request/all")
    assert res.status_code == 200
    response = res.json()
    assert len(response) == 2
    expected = [
        {
            "request": {
                "scheme": "http",
                "host": "mender-device-auth",
                "port": "8080",
                "method": "GET",
                "path": "/api/internal/v1/devauth/tenants/"
                + tenant_id
                + "/devices/"
                + device_id
                + "/status",
                "queryStringParameters": {},
                "fragment": "",
                "headers": {
                    "Accept-Encoding": ["gzip"],
                    "User-Agent": ["Go-http-client/1.1"],
                    "X-Men-Requestid": [request_id],
                },
                "cookies": {},
                "body": "",
            },
        },
        {
            "request": {
                "scheme": "http",
                "host": "mender-inventory",
                "port": "8080",
                "method": "POST",
                "path": "/api/internal/v1/inventory/tenants/"
                + tenant_id
                + "/devices/"
                + device_status,
                "queryStringParameters": {},
                "fragment": "",
                "headers": {
                    "Content-Type": ["application/json"],
                    "Accept-Encoding": ["gzip"],
                    "Content-Length": ["5"],
                    "User-Agent": ["Go-http-client/1.1"],
                    "X-Men-Requestid": [request_id],
                },
                "cookies": {},
                "body": '["' + device_id + '"]',
            },
        },
    ]
    assert expected[0]["request"] == response[0]["request"]
    assert expected[1]["request"] == response[1]["request"]
