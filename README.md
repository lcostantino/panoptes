# Panoptes


<img src="/monster.jpg" width="512" height="512">

## Description

Panoptes is just a simple ETW wrapper that allows to subscribe to different providers thru config files and post-process the events using duktape JS runtime

Events are serialized to JSON


## Config File

|Field|Type|Value|
|---|---|---|
|name|string|A name to identify the provider once the event is fired|
|guid|string|A valid GUID provider|
|disabled|bool|Enable/Disable subscription|
|report|string|Only json is valid today|
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
```


## JS Processor

Just define a function named "panoptesProcess" that will receive a JSON String with the full ETW object serialized.

To print data to stdout just return a string here or use console.log. 
To STOP printing the msg outside of JS or if you want to filter the EVENT just return an EMPTY string.
```js
function panoptesProcess(jsonData) {
    jsObject = JSON.parse(jsonData)
    jsObject["modifiedFromJS"] = 1
    //return empty string to avoid logging the event
    return  JSON.stringify(jsObject)
}
```
