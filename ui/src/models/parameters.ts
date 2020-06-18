/*
 * eslint-disable
 */

import * as msRest from "@azure/ms-rest-js";

export const key: msRest.OperationURLParameter = {
  parameterPath: "key",
  mapper: {
    required: true,
    serializedName: "key",
    type: {
      name: "String"
    }
  }
};
export const value: msRest.OperationQueryParameter = {
  parameterPath: [
    "options",
    "value"
  ],
  mapper: {
    serializedName: "value",
    type: {
      name: "String"
    }
  }
};
