import React, { FunctionComponent, useState } from 'react';
import logo from './logo.svg';
import { Layout, Menu, Breadcrumb, Input, Button } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined } from '@ant-design/icons';
// import * as scp from 'scale-color-perceptual';
import './App.css';

const { Header, Content, Footer } = Layout;
const { TextArea } = Input;

interface DbPageProps {
  searchTerm: string;
  searchResult: string;
  // clickHandler: (event: React.SyntheticEvent<KeyboardEvent>) => void;
  clickHandler: React.KeyboardEventHandler<HTMLInputElement>;
}

function DatabasePage(props: DbPageProps) {
  // Add db search functionality (how best to ensure that searching displays
  // results, but does not clear the input field?)

  return (
    <>
      <Input className="App-key-input" placeholder="Search for a key here..." onPressEnter={props.clickHandler} allowClear />
      <br />
      <br />
      <TextArea className="App-result-field" value={props.searchResult} allowClear />
      <br />
      <br />
      <Button className="db-button" id="search-button"><SearchOutlined /> Search</Button>
      <Button className="db-button" id="copy-button"><CopyOutlined /> Copy to clipboard</Button>
      <Button className="db-button" id="save-button"><SaveOutlined /> Save to database</Button>
      <Button className="db-button" id="delete-button"><DeleteOutlined /> Delete from database</Button>
      <br />
      <br />
    </>
  )
}

function AdminPage() {
  return (
    <>
      Nothing here yet
    </>
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
}

function AppContent(props:AppContentProps) {
  const pages: Record<Page, JSX.Element> = {
    Home: <HomePage />,
    Database: <DatabasePage searchTerm={props.searchTerm} searchResult={props.searchResult} clickHandler={props.searchHandler}/>,
    Admin: <AdminPage />
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
  // Get currently selected header tab and send it to AppContent props
  // to choose which page content to display

  // const colors = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1].map(scp.magma);
  // colors.forEach((i) => {console.log(i)});

  const searchHandler: React.KeyboardEventHandler<HTMLInputElement> = (event: React.KeyboardEvent<HTMLInputElement>) => {
    let target = event.target as HTMLInputElement;
    setCurrentSearch(target.value);

    const query = `http://localhost:8080/db/${target.value}`
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
      />
      <AppFooter />
    </Layout>
  );
}

export default App;
