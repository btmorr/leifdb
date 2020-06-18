/*
 * MIT
 */

import * as msRest from "@azure/ms-rest-js";
import * as Models from "./models";

const packageName = "leifdb";
const packageVersion = "0.1.0-beta.2+use-autorest.dc53421";

export class LeifDbClientAPIContext extends msRest.ServiceClient {

  /**
   * Initializes a new instance of the LeifDbClientAPIContext class.
   * @param [options] The parameter options
   */
  constructor(options?: Models.LeifDbClientAPIOptions) {
    if (!options) {
      options = {};
    }

    if (!options.userAgent) {
      const defaultUserAgent = msRest.getDefaultUserAgentValue();
      options.userAgent = `${packageName}/${packageVersion} ${defaultUserAgent}`;
    }

    super(undefined, options);

    this.baseUri = options.baseUri || this.baseUri || "http://localhost";
    this.requestContentType = "application/json; charset=utf-8";
  }
}
