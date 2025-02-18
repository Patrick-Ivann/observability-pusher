Expected dictionary in the ~/.obs-pusher/dictionary.xml 

go run main.go events push --element dummy --event-id=MESSAGE.ONE --message=poubelle --namespace="testing" --interval=10 --pod-labels=bip:boup


go run main.go metrics push --namespace=testing --element=dummy --metric=test_metric --value=4 --label=test-labe --pod-labels=ap-name:dummy
go run main.go events list



sequence example 

```
    {
        "ID": "1",
        "values": ["golang","arm"],
        "labels": ["app-ap:mgf","obs-pusher:events"],
        "name": "gogo",
        "repetition": 1,
        "interval": 10
    },
    {
        "ID": "1",
        "values": ["rust","amd"],
        "name": "gogo",
        "repetition": 2,
        "interval": 5
    },
    {
        "ID": "2",
        "values": ["software"],
        "name": "vroom",
        "repetition": 1,
        "interval": 2
    }
```

```
<dictionary xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<metric name="application.part.proctime" fullyQualifiedName="application_part_proctime" type="Timer" description="Time to process" tags="message=file,id" />
</dictionary>
```