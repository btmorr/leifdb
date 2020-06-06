import React from 'react';
import logo from './logo.svg';
import 'bootstrap/dist/css/bootstrap.min.css';
import './App.css';
import Button from 'react-bootstrap/Button';
import Navbar from 'react-bootstrap/Navbar';
import Nav from 'react-bootstrap/Nav';
import InputGroup from 'react-bootstrap/InputGroup';
import FormControl from 'react-bootstrap/FormControl';
// import * as scp from 'scale-color-perceptual';

function Header() {
  const ghSheildImg = "https://img.shields.io/github/stars/btmorr/leifdb?style=social"
  const ghRepoLink = "https://github.com/btmorr/leifdb"
  const ghLinkAlt = "View LeifDb on GitHub"

  return (
    <span className="App-header">
      <Navbar expand="sm" className="App-nav">
        <img src={logo} className="App-logo" alt="logo" />
        <Navbar.Brand href="#home">
          <span className="App-brand">
            LeifDb
          </span>
        </Navbar.Brand>
        <Navbar.Toggle aria-controls="basic-navbar-nav" />
        <Navbar.Collapse id="basic-navbar-nav">
          <Nav className="mr-auto">
            <Nav.Link
              className="App-nav-link"
              id="admin" href="#admin"
            >
              Admin
            </Nav.Link>
            <Nav.Link
              className="App-nav-link"
              id="database" href="#database"
            >
              Database
            </Nav.Link>
          </Nav>
          <Button
            className="App-button"
            href={ghRepoLink}
          >
            <img src={ghSheildImg} alt={ghLinkAlt} />
          </Button>
        </Navbar.Collapse>
      </Navbar>
    </span>
  )
}

function Body() {
  // how to get the text out of the input bar?
  const results = '{"static": "replace this with the fetch results"}';

  return (
    <span className="App-body">
      <span className="App-search-container">
        <InputGroup className="input-group-0" id="search-key-bar">
          <InputGroup.Prepend>
            <InputGroup.Text id="key-input-label">Key</InputGroup.Text>
          </InputGroup.Prepend>
          <FormControl
            placeholder="Enter your search term..."
            aria-label="Key"
            aria-describedby="key-input-label"
          />
        </InputGroup>
        <InputGroup className="input-group-0" id="search-result-bar">
          <FormControl as="textarea" aria-label="textarea" readOnly>{results}</FormControl>
        </InputGroup>
      </span>
    </span>
  )
}

function SideColumn() {
  return (
    <span className="App-side-column" />
  )
}

function Footer() {
  return (
    <span className="App-footer">Copyright 2020, the LeifDb developers</span>
  )
}

function App() {
  // const colors = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1].map(scp.magma);
  // colors.forEach((i) => {console.log(i)});

  // Hooks for which tab is selected in the header? Need to create the components
  // for the Admin page and select off of the active-tab state. (is this even the
  // right way to think about this?)

  return (
    <div className="App">
      <Header />
      <SideColumn />
      <Body />
      <Footer />
    </div>
  );
}

export default App;
