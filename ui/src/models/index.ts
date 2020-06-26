/* eslint-disable */


import { ServiceClientOptions } from "@azure/ms-rest-js";
import * as msRest from "@azure/ms-rest-js";

/**
 * An interface representing MainDeleteResponse.
 */
export interface MainDeleteResponse {
  status?: string;
}

/**
 * An interface representing MainHealthResponse.
 */
export interface MainHealthResponse {
  status?: string;
}

/**
 * An interface representing MainReadResponse.
 */
export interface MainReadResponse {
  value?: string;
}

/**
 * An interface representing MainWriteRequest.
 */
export interface MainWriteRequest {
  value?: string;
}

/**
 * An interface representing MainWriteResponse.
 */
export interface MainWriteResponse {
  status?: string;
}

/**
 * An interface representing LeifDbClientAPIOptions.
 */
export interface LeifDbClientAPIOptions extends ServiceClientOptions {
  baseUri?: string;
}

/**
 * Defines headers for db-write operation.
 */
export interface DbWriteHeaders {
  /**
   * Redirect address of the current leader
   */
  location: string;
}

/**
 * Defines headers for db-delete operation.
 */
export interface DbDeleteHeaders {
  /**
   * Redirect address of current leader
   */
  location: string;
}

/**
 * Contains response data for the dbRead operation.
 */
export type DbReadResponse = MainReadResponse & {
  /**
   * The underlying HTTP response.
   */
  _response: msRest.HttpResponse & {
      /**
       * The response body as text (string format)
       */
      bodyAsText: string;

      /**
       * The response body as parsed JSON or XML
       */
      parsedBody: MainReadResponse;
    };
};

/**
 * Contains response data for the dbWrite operation.
 */
export type DbWriteResponse = DbWriteHeaders & {
  /**
   * The parsed response body.
   */
  body: any;

  /**
   * The underlying HTTP response.
   */
  _response: msRest.HttpResponse & {
      /**
       * The parsed HTTP response headers.
       */
      parsedHeaders: DbWriteHeaders;

      /**
       * The response body as text (string format)
       */
      bodyAsText: string;

      /**
       * The response body as parsed JSON or XML
       */
      parsedBody: any;
    };
};

/**
 * Contains response data for the dbDelete operation.
 */
export type DbDeleteResponse = DbDeleteHeaders & {
  /**
   * The parsed response body.
   */
  body: any;

  /**
   * The underlying HTTP response.
   */
  _response: msRest.HttpResponse & {
      /**
       * The parsed HTTP response headers.
       */
      parsedHeaders: DbDeleteHeaders;

      /**
       * The response body as text (string format)
       */
      bodyAsText: string;

      /**
       * The response body as parsed JSON or XML
       */
      parsedBody: any;
    };
};

/**
 * Contains response data for the httpHealth operation.
 */
export type HttpHealthResponse = MainHealthResponse & {
  /**
   * The underlying HTTP response.
   */
  _response: msRest.HttpResponse & {
      /**
       * The response body as text (string format)
       */
      bodyAsText: string;

      /**
       * The response body as parsed JSON or XML
       */
      parsedBody: MainHealthResponse;
    };
};
