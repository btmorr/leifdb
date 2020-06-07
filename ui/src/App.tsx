import React, { FunctionComponent, useState } from 'react';
import logo from './logo.svg';
import { Layout, Menu, Breadcrumb, Input, Button, Space, Alert, Card } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined, CheckCircleFilled, WarningFilled } from '@ant-design/icons';
// import * as scp from 'scale-color-perceptual';
import './App.css';

const { Header, Content, Footer } = Layout;
const { TextArea, Search } = Input;

interface DbPageProps {
  host: Server;
}

function DatabasePage(props: DbPageProps) {
  const [searchResult, setSearchResult] = useState("");

  function connectHeader() {
    if (props.host.address) {
      return (
        <span>{props.host.healthy ? <span><CheckCircleFilled /> Connected to</span> : <span><WarningFilled /> Could not connect to</span>} <code>{props.host.address}</code></span>
      )
    }
    return (
      <Alert type="error" message="Not connected to a database server--use the Admin tab to connect" showIcon />
    )
  }

  const searchHandler: React.KeyboardEventHandler<HTMLInputElement> = (event: React.KeyboardEvent<HTMLInputElement>) => {
    let target = event.target as HTMLInputElement;
    const query = `http://${props.host.address}/db/${target.value}`
    console.log("GET " + query)
    fetch(query)
      .then(response => response.text())
      .then(data => {
        console.log("Result:", data);
        setSearchResult(data);
      });
  }

  return (
    <Space direction="vertical">
      {connectHeader()}
      <Input
        className="App-key-input"
        placeholder="Search for a key here..."
        onPressEnter={searchHandler}
        allowClear
      />
      <TextArea
        className="App-result-field"
        value={searchResult}
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
  currentHost: Server;
  setHost: React.Dispatch<React.SetStateAction<Server>>;
}

function AdminPage(props:AdminPageProps) {

  function tryConnect(address: string, handler: React.Dispatch<React.SetStateAction<Server>>) {
    const query = `http://${address}/health`
    fetch(query)
      .then(res => {
        if (!res.ok) {
          throw Error(res.statusText);
        }
        return res;
      })
      .then(res => handler({address: address, healthy: res.ok}))
      .catch(() => handler({address: address, healthy: false}));
  }

  function buildCard() {
    if (props.currentHost.address) {
      return (
        <div className="site-card-border-less-wrapper">
          <Card title="LeifDb server" size="small" style={{ width: 300 }}>
            <p>Host: <code>{props.currentHost.address}</code></p>
            <p>Status: {props.currentHost.healthy ? <span><CheckCircleFilled /> Connected</span> : <span><WarningFilled /> Could not connect</span>}</p>
          </Card>
        </div>
      )
    }
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
      {buildCard()}
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
}

interface Server {
  address: string;
  healthy: boolean;
}

function AppContent(props:AppContentProps) {
  const [host, setHost] = useState({address: "", healthy: false});

  const pages: Record<Page, JSX.Element> = {
    Home: <HomePage />,
    Database: <DatabasePage
      host={host}/>,
    Admin: <AdminPage
      currentHost={host}
      setHost={setHost}/>
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

const App:FunctionComponent<{ initialPage?: Page }> = ({ initialPage = "Admin" }) => {
  const [currentPage, setCurrentPage] = useState(initialPage);
  // Get currently selected header tab and send it to AppContent props
  // to choose which page content to display

  // const colors = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1].map(scp.magma);
  // colors.forEach((i) => {console.log(i)});

  return (
    <Layout className="layout">
      <AppHeader
        page={currentPage}
        clickHandler={setCurrentPage}
      />
      <AppContent
        page={currentPage}
      />
      <AppFooter />
    </Layout>
  );
}

export default App;
