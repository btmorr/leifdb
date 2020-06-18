/*
 * MIT
 */

import * as msRest from "@azure/ms-rest-js";


export const MainDeleteResponse: msRest.CompositeMapper = {
  serializedName: "main.DeleteResponse",
  type: {
    name: "Composite",
    className: "MainDeleteResponse",
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

export const MainReadResponse: msRest.CompositeMapper = {
  serializedName: "main.ReadResponse",
  type: {
    name: "Composite",
    className: "MainReadResponse",
    modelProperties: {
      value: {
        serializedName: "value",
        type: {
          name: "String"
        }
      }
    }
  }
};

export const MainWriteRequest: msRest.CompositeMapper = {
  serializedName: "main.WriteRequest",
  type: {
    name: "Composite",
    className: "MainWriteRequest",
    modelProperties: {
      value: {
        serializedName: "value",
        type: {
          name: "String"
        }
      }
    }
  }
};

export const MainWriteResponse: msRest.CompositeMapper = {
  serializedName: "main.WriteResponse",
  type: {
    name: "Composite",
    className: "MainWriteResponse",
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
