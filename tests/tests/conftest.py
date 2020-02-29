import pytest
import requests

MMOCK_URI = "http://mmock:8082"
WORKFLOWS_URI = "http://workflows-server:8080"


@pytest.fixture(scope="session")
def workflows_url():
    yield WORKFLOWS_URI


@pytest.fixture(scope="function")
def mmock_url():
    res = requests.get(MMOCK_URI + "/api/request/reset")
    assert res.status_code == 200
    yield MMOCK_URI
