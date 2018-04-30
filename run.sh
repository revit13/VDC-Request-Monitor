#!/bin/bash 
echo "starting monitoring proxy for ${VDC_NAME}"
sleep 10
# create an ingest node in the ES to map the logdata send by filebeat
curl -H 'Content-Type: application/json' -XPUT "http://${elasticURI}/_ingest/pipeline/nginx-pipeline" -d@pipeline.json

# subsitute the env set in the docker-compose file

envsubst '${VDC_ADDRESS},${VDC_PORT},${OPENTRACING}' < /nginx.conf > /etc/nginx/nginx.conf
envsubst '${VDC_NAME},${elasticURI}' < /etc/filebeat/filebeat.yml > /etc/filebeat/filebeat.yml

#start filebeat and nginx
exec  service filebeat start & nginx -g "daemon off;" 
