import React from "react";
import { render, unmountComponentAtNode } from "react-dom";
import { act } from "react-dom/test-utils";
import AdminPage from "../AppContent/AdminPage";

let container = null;
beforeEach(() => {
  // setup a DOM element as a render target
  container = document.createElement("div");
  document.body.appendChild(container);
});

afterEach(() => {
  // cleanup on exiting
  unmountComponentAtNode(container);
  container.remove();
  container = null;
});

it("shows success card when connected to a server", async () => {
  jest.spyOn(global, "fetch").mockImplementation(() => Promise.resolve({
    text: () => Promise.resolve("Ok")
  }));

  let host = {
      address: "localhost:1234",
      healthy: true
  };

  await act(async () => {
    render(<AdminPage currentHost={host} setHost={() => {}}/>, container);
  });

  expect(container.textContent).toContain("Connected");

  global.fetch.mockRestore();
});

it("shows failure card when not connected to a server", async () => {
  jest.spyOn(global, "fetch").mockImplementation(() => Promise.resolve({
    text: () => Promise.resolve("Ok")
  }));

  let host = {
      address: "localhost:1234",
      healthy: false
  };

  await act(async () => {
    render(<AdminPage currentHost={host} setHost={() => {}}/>, container);
  });

  expect(container.textContent).toContain("Could not connect");

  global.fetch.mockRestore();
});
