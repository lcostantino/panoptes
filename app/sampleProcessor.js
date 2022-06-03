

function panoptesProcess(jsonData) {

    console.log(jsonData)
    jsObject = JSON.parse(jsonData)
    jsObject["modifiedFromJS"] = 1
    
    //return empty string to avoid logging the event
    return  JSON.stringify(jsObject)
}