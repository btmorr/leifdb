/*
 * eslint-disable
 */

import * as msRest from "@azure/ms-rest-js";


export const MainHealthResponse: msRest.CompositeMapper = {
  serializedName: "main.HealthResponse",
  type: {
    name: "Composite",
    className: "MainHealthResponse",
    modelProperties: {
      status: {
        serializedName: "status",
        type: {
          name: "String"
        }
      }
    }
  }
};

export const DbWriteHeaders: msRest.CompositeMapper = {
  serializedName: "db-write-headers",
  type: {
    name: "Composite",
    className: "DbWriteHeaders",
    modelProperties: {
      location: {
        serializedName: "location",
        type: {
          name: "String"
        }
      }
    }
  }
};

export const DbDeleteHeaders: msRest.CompositeMapper = {
  serializedName: "db-delete-headers",
  type: {
    name: "Composite",
    className: "DbDeleteHeaders",
    modelProperties: {
      location: {
        serializedName: "location",
        type: {
          name: "String"
        }
      }
    }
  }
};
