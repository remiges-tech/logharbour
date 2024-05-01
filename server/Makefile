build:
	cd elasticSearchCtl/; go build -o elasticSearchCtl

create-index:
	cd elasticSearchCtl/; ./elasticSearchCtl create logharbour

add-logs:
	cd elasticSearchCtl/; ./elasticSearchCtl add ./log4.json logharbour

etcd:
	cd ; cd etcd/bin; ./etcd

all-es:
  	cd elasticSearchCtl/; go build -o elasticSearchCtl; ./elasticSearchCtl create logharbour; ./elasticSearchCtl add ./logs.json;
.PHONY: build create-index all
