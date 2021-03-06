# Myriag

Arbitrary code execution server using Docker //in Go//.

## Inspiration
- [Myriad](https://github.com/1Computer1/myriad)
- [Myrias](https://github.com/iCrawl/myrias)

## Why 
 Just to play around with Docker SDK

## Endpoints

### **GET** `/languages`
List of enabled languages.  
Example response:

```json
["go", "typescript"]
```

### **POST** `/eval`
Evaluate code.  
JSON payload with `language` and `code` keys.  
The `language` is as in the name of a subfolder in the `languages` directory.  
Example payload:

```json
{ "language": "go", "code": "package main; import \"fmt\"; func main() { fmt.Println(\"hello world\")}" }
```

Example response:
```json
{ "result": "hello world\n" }
```

Errors with 404 if `language` is not found, `504` if evaluation timed out, or `500` if evaluation failed for other reasons.

### **GET** `/containers`
List of containers being handled by Myriag.

### **POST** `/cleanup`
Kill all containers, giving back the names of the containers killed.