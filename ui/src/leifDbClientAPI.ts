import * as msRest from "@azure/ms-rest-js";
import * as Models from "./models";
import * as Mappers from "./models/mappers";
import * as Parameters from "./models/parameters";
import { LeifDbClientAPIContext } from "./leifDbClientAPIContext";

class LeifDbClientAPI extends LeifDbClientAPIContext {
  /**
   * Initializes a new instance of the LeifDbClientAPI class.
   * @param [options] The parameter options
   */
  constructor(options?: Models.LeifDbClientAPIOptions) {
    super(options);
  }

  /**
   * @summary Return value from database by key
   * @param key Key
   * @param [options] The optional parameters
   * @returns Promise<Models.DbReadResponse>
   */
  dbRead(key: string, options?: msRest.RequestOptionsBase): Promise<Models.DbReadResponse>;
  /**
   * @param key Key
   * @param callback The callback
   */
  dbRead(key: string, callback: msRest.ServiceCallback<Models.MainReadResponse>): void;
  /**
   * @param key Key
   * @param options The optional parameters
   * @param callback The callback
   */
  dbRead(key: string, options: msRest.RequestOptionsBase, callback: msRest.ServiceCallback<Models.MainReadResponse>): void;
  dbRead(key: string, options?: msRest.RequestOptionsBase | msRest.ServiceCallback<Models.MainReadResponse>, callback?: msRest.ServiceCallback<Models.MainReadResponse>): Promise<Models.DbReadResponse> {
    return this.sendOperationRequest(
      {
        key,
        options
      },
      dbReadOperationSpec,
      callback) as Promise<Models.DbReadResponse>;
  }

  /**
   * @summary Write value to database by key
   * @param key Key
   * @param body Value
   * @param [options] The optional parameters
   * @returns Promise<Models.DbWriteResponse>
   */
  dbWrite(key: string, body: Models.MainWriteRequest, options?: msRest.RequestOptionsBase): Promise<Models.DbWriteResponse>;
  /**
   * @param key Key
   * @param body Value
   * @param callback The callback
   */
  dbWrite(key: string, body: Models.MainWriteRequest, callback: msRest.ServiceCallback<any>): void;
  /**
   * @param key Key
   * @param body Value
   * @param options The optional parameters
   * @param callback The callback
   */
  dbWrite(key: string, body: Models.MainWriteRequest, options: msRest.RequestOptionsBase, callback: msRest.ServiceCallback<any>): void;
  dbWrite(key: string, body: Models.MainWriteRequest, options?: msRest.RequestOptionsBase | msRest.ServiceCallback<any>, callback?: msRest.ServiceCallback<any>): Promise<Models.DbWriteResponse> {
    return this.sendOperationRequest(
      {
        key,
        body,
        options
      },
      dbWriteOperationSpec,
      callback) as Promise<Models.DbWriteResponse>;
  }

  /**
   * @summary Delete item from database by key
   * @param key Key
   * @param [options] The optional parameters
   * @returns Promise<Models.DbDeleteResponse>
   */
  dbDelete(key: string, options?: msRest.RequestOptionsBase): Promise<Models.DbDeleteResponse>;
  /**
   * @param key Key
   * @param callback The callback
   */
  dbDelete(key: string, callback: msRest.ServiceCallback<any>): void;
  /**
   * @param key Key
   * @param options The optional parameters
   * @param callback The callback
   */
  dbDelete(key: string, options: msRest.RequestOptionsBase, callback: msRest.ServiceCallback<any>): void;
  dbDelete(key: string, options?: msRest.RequestOptionsBase | msRest.ServiceCallback<any>, callback?: msRest.ServiceCallback<any>): Promise<Models.DbDeleteResponse> {
    return this.sendOperationRequest(
      {
        key,
        options
      },
      dbDeleteOperationSpec,
      callback) as Promise<Models.DbDeleteResponse>;
  }

  /**
   * @summary Return server health status
   * @param [options] The optional parameters
   * @returns Promise<Models.HttpHealthResponse>
   */
  httpHealth(options?: msRest.RequestOptionsBase): Promise<Models.HttpHealthResponse>;
  /**
   * @param callback The callback
   */
  httpHealth(callback: msRest.ServiceCallback<Models.MainHealthResponse>): void;
  /**
   * @param options The optional parameters
   * @param callback The callback
   */
  httpHealth(options: msRest.RequestOptionsBase, callback: msRest.ServiceCallback<Models.MainHealthResponse>): void;
  httpHealth(options?: msRest.RequestOptionsBase | msRest.ServiceCallback<Models.MainHealthResponse>, callback?: msRest.ServiceCallback<Models.MainHealthResponse>): Promise<Models.HttpHealthResponse> {
    return this.sendOperationRequest(
      {
        options
      },
      httpHealthOperationSpec,
      callback) as Promise<Models.HttpHealthResponse>;
  }
}

// Operation Specifications
const serializer = new msRest.Serializer(Mappers);
const dbReadOperationSpec: msRest.OperationSpec = {
  httpMethod: "GET",
  path: "db/{key}",
  urlParameters: [
    Parameters.key
  ],
  responses: {
    200: {
      bodyMapper: Mappers.MainReadResponse
    },
    default: {}
  },
  serializer
};

const dbWriteOperationSpec: msRest.OperationSpec = {
  httpMethod: "PUT",
  path: "db/{key}",
  urlParameters: [
    Parameters.key
  ],
  requestBody: {
    parameterPath: "body",
    mapper: {
      ...Mappers.MainWriteRequest,
      required: true
    }
  },
  responses: {
    200: {
      bodyMapper: Mappers.MainWriteResponse,
      headersMapper: Mappers.DbWriteHeaders
    },
    307: {
      bodyMapper: {
        serializedName: "parsedResponse",
        type: {
          name: "String"
        }
      },
      headersMapper: Mappers.DbWriteHeaders
    },
    400: {
      bodyMapper: {
        serializedName: "parsedResponse",
        type: {
          name: "String"
        }
      },
      headersMapper: Mappers.DbWriteHeaders
    },
    default: {}
  },
  serializer
};

const dbDeleteOperationSpec: msRest.OperationSpec = {
  httpMethod: "DELETE",
  path: "db/{key}",
  urlParameters: [
    Parameters.key
  ],
  responses: {
    200: {
      bodyMapper: Mappers.MainDeleteResponse,
      headersMapper: Mappers.DbDeleteHeaders
    },
    307: {
      bodyMapper: {
        serializedName: "parsedResponse",
        type: {
          name: "String"
        }
      },
      headersMapper: Mappers.DbDeleteHeaders
    },
    default: {}
  },
  serializer
};

const httpHealthOperationSpec: msRest.OperationSpec = {
  httpMethod: "GET",
  path: "health",
  responses: {
    200: {
      bodyMapper: Mappers.MainHealthResponse
    },
    default: {}
  },
  serializer
};

export {
  LeifDbClientAPI,
  LeifDbClientAPIContext,
  Models as LeifDbModels,
  Mappers as LeifDbMappers
};
