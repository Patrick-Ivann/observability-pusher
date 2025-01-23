Expected dictionary in the ~/.obs-pusher/dictionary.xml 

go run main.go events push --element dummy --event-id=MESSAGE.ONE --message=poubelle --namespace="testing" --interval=10 --pod-labels=bip:boup


go run main.go metrics push --namespace=testing --element=dummy --metric=test_metric --value=4 --label=test-labe --pod-labels=ap-name:dummy
go run main.go events list