build:
	cd elasticSearchCtl/; go build -o elasticSearchCtl

create-index:
	cd elasticSearchCtl/; ./elasticSearchCtl create logharbour

add-logs:
	cd elasticSearchCtl/; ./elasticSearchCtl add ./logs.json

all:
  	cd elasticSearchCtl/; go build -o elasticSearchCtl; ./elasticSearchCtl create logharbour; ./elasticSearchCtl add ./logs.json;