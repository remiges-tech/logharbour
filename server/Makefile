build:
	go build -o elasticCLI

create-index:
	 cd elasticSearchCtl/; ./elasticCLI create logharbour

add-logs:
	cd elasticSearchCtl/; ./elasticCLI add ./logs.json

all:
  cd elasticSearchCtl/; go build -o elasticCLI; ./elasticCLI create logharbour; ./elasticCLI add ./logs.json;