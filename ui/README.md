# LeifDb UI


🚧 **Under heavy construction -- contributions welcome!** 🚧

This project provides a simple front end for interacting with a LeifDb server.

Currently, it is entirely static. Current component breakdown is open to revision. Desired functionality:
- tabs on the nav-bar should render different body sections
- the database tab should have a search bar with an input that lets the user enter a key, and it should query the server and return the value for that key in the result text field, and provide buttons for search/copy/save/delete
- the admin tab should query the server for its configuration, and then render a card for each and send health checks to display their status (server will have to be modified to provide configuration info)

## Install

Ensure that `yarn` is installed, along with the most recent LTS version of Node.js. Then, in this directory run:

```
yarn
```

## Available Scripts

In this directory, you can run:

### `yarn start`

Runs the app in the development mode.
Open [http://localhost:3000](http://localhost:3000) to view it in the browser.

The page will reload if you make edits, and you will see any lint errors in the console.

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
