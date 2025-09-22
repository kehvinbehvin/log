# Braintrust API Structure Verification

## Current Implementation Analysis

### Function Call Structure
- **Function ID**: `"a26dfd04-0fd7-4a77-aa45-826560d785ab"`
- **Method**: `sc.bClient.Functions.Invoke()`
- **Parameters**: `braintrust.FunctionInvokeParams`

### Request Structure
```go
braintrust.FunctionInvokeParams{
    Input: map[string]interface{}{
        "examples": "<examples><i>03-17 16:13:45.382  1702  3697 D PowerManagerService: acquire lock=189667585, flags=0x1, tag=\"*launch*\", name=android, ws=WorkSource{10113}, uid=1000, pid=1702</i></examples>",
        "template": "<template>Y-Y Y:Y:Y.Y  Y  Y Y Y: Y Y=Y, Y=Y, Y=\"X\", Y=Y, Y=Y{X}, Y=Y, Y=Y</template>",
    },
}
```

### Expected Response Structure
```go
type ContextualiseResponse struct {
    Labels []string `json:"labels"`
}
```

### Response Processing
- API returns `string` response
- Response is JSON parsed using `json.Unmarshal([]byte(response), &contextResponse)`
- Final Context created: `Context{labels: contextResponse.Labels}`

## Notes for Testing
1. The API expects hardcoded examples and template format
2. Response is expected to be JSON with a "labels" array
3. Error handling includes both API errors and JSON parsing errors
4. The function ID is static and specific to this use case

## Test Mock Requirements
- Mock HTTP responses should return JSON: `{"labels": ["label1", "label2", ...]}`
- Test various label arrays for different scenarios
- Test malformed JSON responses for error handling
- Test API errors (network, auth, etc.)