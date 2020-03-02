FROM python:3.8-slim

RUN pip3 install pytest requests

COPY /tests/* /testing/

CMD ["/testing/run.sh"]
