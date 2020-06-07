# LeifDb UI

🚧 **Under heavy construction -- contributions welcome!** 🚧

This project provides a simple front end for interacting with a LeifDb server.

Currently, search works by entering a term into the text input bar and hitting 'Enter'. Before searching, use the Admin page to enter the address of the HTTP interface of a LeifDb server. The copy and delete buttons on the Database page work ('copy' populates the clipboard with the test of the result field, and 'delete' deletes the current search key from the database if it exists).

The UI does not currently include write functionality. To see the UI in action, you'll need to populate the database with at least one value. The easiest way to do that currently is to launch the server and then use the PUT section of the server's [Swagger page](http://localhost:8080/swagger/index.html) to send at least one key-value pair. Then, you should be able to use the UI to search for the same key.

## Install

Ensure that `yarn` is installed, along with the most recent LTS version of Node.js. Then, in this directory run:

```
yarn
```

## Available Scripts

In this directory, you can run:

### `yarn start`

Runs the app in the development mode--open [http://localhost:3000](http://localhost:3000) to view it in the browser

The page will reload if you make edits, and you will see any lint errors in the console

### `yarn test`

Launches the test runner in the interactive watch mode

### `yarn build`

Builds the app for production to the `build` folder

____

This project was bootstrapped with [Create React App] and the [Ant Design] component library

[Create React App]: https://github.com/facebook/create-react-app
[Ant Design]: https://ant.design/

<!-- notes below for once the color theme is added

The color palette was based very closely on the "Magma" colormap created by [Stéfan van der Walt] and [Nathaniel J. Smith] for the Python matplotlib project to ensure colorblind accessibility. Development of the exact palette used for this site was aided by [politiken-journalism/scale-color-perceptual]. For more info, check out the [colormap] page.

[Stéfan van der Walt]: https://github.com/stefanv
[Nathaniel J. Smith]: https://github.com/njsmith
[colormap]: http://bids.github.io/colormap/
[politiken-journalism/scale-color-perceptual]: https://github.com/politiken-journalism/scale-color-perceptual
-->
