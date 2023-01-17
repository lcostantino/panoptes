# Panoptes


<img src="/monster.jpg" width="512" height="512">

## Description

Panoptes is just a simple ETW wrapper that allows to subscribe to different providers thru config files and post-process the events using duktape JS runtime

Events are serialized to JSON

## Build

ETW lib depends on cygwin , so x86_64-w64-mingw32-gcc must be installed.

This project uses [GoReleaser](https://goreleaser.com), execute build.sh or build_tmp.sh to compile the project. 




## Config File

|Field|Type|Value|
|---|---|---|
|name|string|A name to identify the provider once the event is fired|
|guid|string|A valid GUID provider|
|disabled|bool|Enable/Disable subscription|
|report|string|Only json is valid today|
|rawData|bool|If true ETW Raw data will be inclued as base64 encoded data at jsObject["RawData"]|
|option.level|number|ETW lvl LogAlways (0x0), Critical (0x1), Error (0x2), Warning (0x3), Information (0x4), erbose (0x5)|
|option.matchAnyKeyword|number|https://docs.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-enabletraceex2|
|option.matchAllKeyword|number|https://docs.microsoft.com/en-us/windows/win32/api/evntrace/nf-evntrace-enabletraceex2|
|option.eventsIds|array of numbers|Events ID to capture|


```json
[
 
    {"name": "dns-query", "guid": "{1C95126E-7EEA-49A9-A3FE-A378B03DDB4D}", "disabled": true, "report": "json", "options": {"level": 5, "matchAnyKeyword": 0, "matchAllKeyword": 0, "eventIds":[]} } 
    
]
```

## Sample Output

```cmd
2022-06-05T17:38:15-03:00 INF Registered guid={22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716} name=kernel

2022-06-05T17:38:16-03:00 INF Data etwEvent={"ExtendedData":{"ActivityID":null,"InstanceInfo":null,"SessionID":null,"StackTrace":null,"UserSID":null},"Header":{"ActivityID":{"Data1":0,"Data2":0,"Data3":0,"Data4":[0,0,0,0,0,0,0,0]},"Channel":16,"Flags":576,"ID":8,"KernelTime":16168,"Keyword":9223372036854775936,"Level":4,"OpCode":0,"ProcessID":1064,"ProcessorTime":195652940218152,"ProviderID":{"Data1":586886358,"Data2":3707,"Data3":16939,"Data4":[160,199,47,173,31,208,231,22]},"Task":8,"ThreadID":1476,"TimeStamp":"2022-06-05T17:38:15.5162358-03:00","UserTime":45554,"Version":0},"Props":{"NewPriority":"16","OldPriority":"15","ProcessID":"4","ThreadID":"560"}} guid={22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716} name=kernel

2022-06-05T17:38:16-03:00 INF Data etwEvent={"ExtendedData":{"ActivityID":null,"InstanceInfo":null,"SessionID":null,"StackTrace":null,"UserSID":null},"Header":{"ActivityID":{"Data1":0,"Data2":0,"Data3":0,"Data4":[0,0,0,0,0,0,0,0]},"Channel":16,"Flags":576,"ID":8,"KernelTime":16168,"Keyword":9223372036854775936,"Level":4,"OpCode":0,"ProcessID":1064,"ProcessorTime":195652940218152,"ProviderID":{"Data1":586886358,"Data2":3707,"Data3":16939,"Data4":[160,199,47,173,31,208,231,22]},"Task":8,"ThreadID":1476,"TimeStamp":"2022-06-05T17:38:15.5164288-03:00","UserTime":45554,"Version":0},"Props":{"NewPriority":"16","OldPriority":"15","ProcessID":"4","ThreadID":"560"}} guid={22fb2cd6-0e7b-422b-a0c7-2fad1fd0e716} name=kernel
```


## Usage
```
Usage of panoptes.exe:
  -config-file string
        Config file for sensors
  -consumers int
        number of consumer routines (default 1)
  -js-file string
        JS processor file
  -log-file string
        Log file
  -stdout
        Print to stdout (default true)
  -stop-file string
        If the file is not present Panoptes will stop
  -no-colors
        Disable colors
   -http-endpoint string
        If not empty will host an HTTP server to retrieve data. Ex: localhost:3999   

```


## JS Processor

Just define a function named "panoptesProcess" that will receive a JSON String with the full ETW object serialized.

To print data to stdout just return a string here or use console.log. 
To STOP printing the msg outside of JS or if you want to filter the EVENT just return an EMPTY string.

```js
function panoptesProcess(jsonData) {
    jsObject = JSON.parse(jsonData)
    rawData = jsObject["RawData"] //ONLY IF ENABLED BY CONFIG rawData: true
    jsObject["modifiedFromJS"] = 1
    //return empty string to avoid logging the event
    return  JSON.stringify(jsObject)
}
```

### Special JS Bindings

```
func pLog(dstFile, str) -> Log to a specific file, ex: pLog("C:\\tmp\\a.txt", "Weird Process found")

func convertStringToUtf8(bytes, start_pos, end_post, utf16 encoding) -> convertStringToUtf8(service_event, 14, idx, "utf-16le")
```
Duktape bindings can also be accessed via Duktape object. See (Builtin-Duktape) (https://duktape.org/guide.html#builtin-duktape)


### Parsing ETW Raw Data

It's possible to receive Base64 encoded event raw data and process it using JS. 
An example of how to parse Service Manager Provider is located at https://github.com/lcostantino/panoptes/blob/main/app/processors/serviceRawProcessor.js

```js
function panoptesProcess(jsonData) {

    if (jsonData == "{}") {
        return null;
    }
    jsObject = JSON.parse(jsonData)

    var service_event = Duktape.dec('base64', jsObject["RawData"]); // we don't need split & map like in v8, here we get the array from C

    event_id = service_event[9] << 8 | service_event[8];
    provider_len = (service_event[13] << 8 | service_event[12]) * 2;
    var idx = 14 + provider_len
    provider_name = convertStringToUtf8(service_event, 14, idx, "utf-16le")

    sid_length = service_event[idx]

    idx += sid_length + 2

    var last_start = idx + 2
    //There are easier ways to do this, but i wan't to show how duktape plain buffers can be used for more comples scenarios.
    var strfound = Array();
    for (var i = last_start, nbytes = 0; i < service_event.length; i += 2, nbytes += 2) {

        uniChar = service_event[i + 1] << 8 | service_event[i]
        if (uniChar == 0) {
            strfound.push(convertStringToUtf8(service_event, last_start, last_start + nbytes, "utf-16le"))
            nbytes = 0;
            i += 2;
            last_start = i;

        }
        if (strfound.length == 5) {
            break;
        }
    }
    jsObject["serviceName"] = strfound[0]
    jsObject["servicePath"] = strfound[1]
    jsObject["serviceType"] = strfound[2]
    jsObject["serviceUser"] = strfound[4]
    jsObject["serviceStartType"] = strfound[3]

    return JSON.stringify(jsObject);

}

```


## Http 

When starting panoptes with `-http-endpoint` option you can request captured data by:

  1. curl http://localhost:yourport/getEvents
  2. curl http://localhost:yourport/getLogFile (This require -log-file parameter to return the entire file)

Note: events are kept in memory for getEvents so invoke this url periodically to flush data.


## Panoptes Specific Fields

Each object will include a Panoptes object with a SessionId that will remain the same during execution. (Will change on every app start)
```json
 "Panoptes": {
        "SessionId": "89b76852-97e4-4f9b-baf6-7d71d7e5d6b8"
 }
```
