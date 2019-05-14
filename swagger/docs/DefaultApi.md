# \DefaultApi

All URIs are relative to *http://minesweeper.tulentsev.com*

Method | HTTP request | Description
------------- | ------------- | -------------
[**MovePost**](DefaultApi.md#MovePost) | **Post** /move | 
[**NewgamePost**](DefaultApi.md#NewgamePost) | **Post** /newgame | 


# **MovePost**
> Game MovePost(ctx, moveInfo)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **moveInfo** | [**MoveInfo**](MoveInfo.md)| Data about your move | 

### Return type

[**Game**](game.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **NewgamePost**
> Game NewgamePost(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

[**Game**](game.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

