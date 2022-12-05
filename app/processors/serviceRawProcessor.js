// Example of how to parse binary data

// Provider Config (Note "rawData": true)
//{"name": "scm-service", "guid": "{555908d1-a6d7-4695-8e1e-26931d2012f4}", "disabled": false, "report": "json", "rawData":true, "options": {"level": 5, "matchAnyKeyword": "ffffffffffffffff", "matchAllKeyword": "0", "eventIds":[]} } 
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