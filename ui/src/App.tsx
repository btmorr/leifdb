import React, { FunctionComponent, useState } from 'react';
import logo from './logo.svg';
import { Layout, Menu, Breadcrumb, Input, Button, Space, Alert } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined } from '@ant-design/icons';
// import * as scp from 'scale-color-perceptual';
import './App.css';

const { Header, Content, Footer } = Layout;
const { TextArea, Search } = Input;

interface DbPageProps {
  searchTerm: string;
  searchResult: string;
  host: string;
  // clickHandler: (event: React.SyntheticEvent<KeyboardEvent>) => void;
  clickHandler: React.KeyboardEventHandler<HTMLInputElement>;
}

function DatabasePage(props: DbPageProps) {
  // Add db search functionality (how best to ensure that searching displays
  // results, but does not clear the input field?)

  function connectHeader() {
    if (props.host) {
      return (
        <span>Connected to: <code>{props.host}</code></span>
      )
    }
    return (
      <Alert type="error" message="Not connected to a database server--use the Admin tab to connect" showIcon />
    )
  }

  return (
    <Space direction="vertical">
      {connectHeader()}
      <Input
        className="App-key-input"
        placeholder="Search for a key here..."
        onPressEnter={props.clickHandler}
        allowClear
      />
      <TextArea
        className="App-result-field"
        value={props.searchResult}
        allowClear
      />
      <div>
        <Button className="db-button" id="search-button"><SearchOutlined /> Search</Button>
        <Button className="db-button" id="copy-button"><CopyOutlined /> Copy to clipboard</Button>
        <Button className="db-button" id="save-button"><SaveOutlined /> Save to database</Button>
        <Button className="db-button" id="delete-button"><DeleteOutlined /> Delete from database</Button>
      </div>
    </Space>
  )
}

interface AdminPageProps {
  setHost: React.Dispatch<React.SetStateAction<string>>;
}

function AdminPage(props:AdminPageProps) {

  function tryConnect(address: string, handler: React.Dispatch<React.SetStateAction<string>>) {
    const query = `http://${address}/health`
    fetch(query)
      .then(res => handler(address));
  }

  return (
    <Space direction="vertical">
      Enter the address of a LeifDb server to connect to:
      <Search
        placeholder="localhost:8080"
        onSearch={value => { tryConnect(value, props.setHost) }}
        style={{ width: 300 }}
      />
      <span>
        Address should be in the form "<code>host:port</code>",
        without quote marks (example: <code>192.168.0.3:8080</code>)
      </span>
    </Space>
  )
}

function HomePage() {
  return (
    <>
      Nothing here yet
    </>
  )
}

type Page = 'Home' | 'Database' | 'Admin';

interface AppContentProps {
  page: Page;
  searchTerm: string;
  searchResult: string;
  searchHandler: React.KeyboardEventHandler<HTMLInputElement>;
  host: string;
  setHost: React.Dispatch<React.SetStateAction<string>>;
}

function AppContent(props:AppContentProps) {
  const pages: Record<Page, JSX.Element> = {
    Home: <HomePage />,
    Database: <DatabasePage
      searchTerm={props.searchTerm}
      searchResult={props.searchResult}
      clickHandler={props.searchHandler}
      host={props.host}/>,
    Admin: <AdminPage
      setHost={props.setHost}/>
  };

  return (
    <Content style={{ padding: '0 50px' }}>
      <Breadcrumb style={{ margin: '16px 0' }}>
        <Breadcrumb.Item href="#home">Home</Breadcrumb.Item>
        <Breadcrumb.Item>{props.page}</Breadcrumb.Item>
      </Breadcrumb>
      <div className="site-layout-content">
        {pages[props.page]}
      </div>
    </Content>
  )
}

interface HeaderProps {
  page: Page;
  clickHandler: (p: Page) => void;
}

function AppHeader(props:HeaderProps) {

  return (
    <Header>
      <Menu theme="dark" mode="horizontal" defaultSelectedKeys={[props.page]}>
        <Menu.Item key="logo">
          <img src={logo} className="App-logo" alt="logo" />
          LeifDb
        </Menu.Item>
        <Menu.Item key="Database" onClick={() => props.clickHandler("Database")}>Database</Menu.Item>
        <Menu.Item key="Admin" onClick={() => props.clickHandler("Admin")}>Admin</Menu.Item>
      </Menu>
    </Header>
  )
}


function AppFooter() {
  const ghSheildImg = "https://img.shields.io/github/stars/btmorr/leifdb?style=social";
  const ghRepoLink = "https://github.com/btmorr/leifdb";
  const ghLinkAlt = "View LeifDb on GitHub";

  return (
    <Footer style={{ textAlign: 'left' }}>
      ©2020 the LeifDb developers <a href={ghRepoLink}><img src={ghSheildImg} alt={ghLinkAlt} /></a>
    </Footer>
  )
}

const App:FunctionComponent<{ initialPage?: Page }> = ({ initialPage = "Database" }) => {
  const [currentPage, setCurrentPage] = useState(initialPage);
  const [currentSearch, setCurrentSearch] = useState("");
  const [searchResults, setSearchResults] = useState("");
  const [host, setHost] = useState("");
  // Get currently selected header tab and send it to AppContent props
  // to choose which page content to display

  // const colors = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1].map(scp.magma);
  // colors.forEach((i) => {console.log(i)});

  const searchHandler: React.KeyboardEventHandler<HTMLInputElement> = (event: React.KeyboardEvent<HTMLInputElement>) => {
    let target = event.target as HTMLInputElement;
    setCurrentSearch(target.value);

    const query = `http://${host}/db/${target.value}`
    console.log("GET " + query)
    fetch(query)
      .then(response => response.text())
      .then(data => {
        console.log("Result:", data);
        setSearchResults(data);
      });
  }

  return (
    <Layout className="layout">
      <AppHeader
        page={currentPage}
        clickHandler={setCurrentPage}
      />
      <AppContent
        page={currentPage}
        searchTerm={currentSearch}
        searchResult={searchResults}
        searchHandler={searchHandler}
        host={host}
        setHost={setHost}
      />
      <AppFooter />
    </Layout>
  );
}

export default App;
