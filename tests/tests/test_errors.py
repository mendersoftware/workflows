import requests


def test_error_workflow_not_found(workflows_url):
    res = requests.post(
        workflows_url + "/api/workflow/workflow_does_not_exist",
        json={"key": "value",},
    )
    assert res.status_code == 404


def test_error_payload_is_not_json(workflows_url):
    res = requests.post(
        workflows_url + "/api/workflow/workflow_does_not_exist", data="DUMMY",
    )
    assert res.status_code == 400
    assert res.json() == {
        "error": "Unable to parse the input parameters: invalid character 'D' looking for beginning of value"
    }
