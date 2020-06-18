import React from "react";
import { rest } from "msw";
import { setupServer } from "msw/node";
import { render, fireEvent, screen, waitFor } from "@testing-library/react";
import { act } from "react-dom/test-utils";
import DatabasePage from "../AppContent/DatabasePage";
import * as Client from '../leifDbClientAPI';

const keyText = "test.key";
const resultText = "Testy";
const server = setupServer(
  rest.get(`*/db/${keyText}`, (req, res, ctx) => {
    // console.log("beep");
    return res(ctx.json({"value": resultText}));
  }),
  rest.put(`*/db/${keyText}`, (req, res, ctx) => {
    // console.log("boop");
    return res(ctx.json({"status": "Ok"}));
  })
);

beforeAll(() => server.listen());
afterEach(() => server.resetHandlers());
afterAll(() => server.close());

test("displays results after fetch when connected to a server", async () => {

  let host = {
    address: "localhost:1234",
    healthy: true,
    client: new Client.LeifDbClientAPI({baseUri: "http://localhost:1234"})
  };
  let connected = true;

  const { getByText } = render(<DatabasePage host={host} connected={connected} />);

  act(() => {
    fireEvent.change(
      screen.getByTestId("App-key-input"), {target: {value: keyText}});
  })

  const keyfield = screen.getByTestId("App-key-input");
  expect(keyfield.value).toBe(keyText);

  // fireEvent.keyPress(keyfield, { key: "Enter", charCode: 13 });
  // For some reason, the keyPress dispatch does not cause the component to
  // do fetch, but the click dispatch does (both work when actually using
  // the app in browser)
  await act(async () => {
    fireEvent.click(screen.getAllByLabelText("search")[1]);
    function sleep(ms) {
      return new Promise(resolve => setTimeout(resolve, ms));
    }
    // There has **got** to be a better way to check the DOM after re-render...
    // But if I don't do this then any other means I've found returns early
    // and the component isn't updated.
    await sleep(10);
  });

  const resultField = getByText(resultText);
  expect(resultField.textContent).toBe(resultText);
});

// test("runs correctly-formed put request to server", async () => {
//
//   let host = "localhost:1234";
//   let connected = true;
//
//   const { getByText } = render(<DatabasePage host={host} connected={connected} />);
//
//   fireEvent.change(screen.getByTestId("App-key-input"), {target: {value: keyText}});
//
//   const keyfield = screen.getByTestId("App-key-input");
//   expect(keyfield.value).toBe(keyText);
//   function sleep(ms) {
//     return new Promise(resolve => setTimeout(resolve, ms));
//   }
//
//   await act(async () => {
//     // This should switch the handler to do a PUT instead of GET
//     fireEvent.change(screen.getByRole("combobox"), { target: { value: "ModeSet" } });
//     await sleep(5);
//   });
//
//   console.log("Combobox value:", screen.getByRole("combobox").value);
//
//   await act(async () => {
//     fireEvent.click(screen.getAllByLabelText("search")[1]);
//     await sleep(5);
//   });
//
//   const resultField = getByText(resultText);
//   expect(resultField.textContent).toBe(resultText);
// });
