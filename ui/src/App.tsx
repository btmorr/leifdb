import React from 'react';
import logo from './logo.svg';
import { Layout, Menu, Breadcrumb, Input, Button } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined } from '@ant-design/icons';
// import * as scp from 'scale-color-perceptual';
import './App.css';

const { Header, Content, Footer } = Layout;
const { TextArea } = Input;

function DatabasePage() {
  // Add db search functionality (how best to ensure that searching displays
  // results, but does not clear the input field?)
  return (
    <>
      <Input className="App-key-input" placeholder="Search for a key here..." allowClear />
      <br />
      <br />
      <TextArea className="App-result-field" allowClear />
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

function AppContent() {
  const page = "Database";

  return (
    <Content style={{ padding: '0 50px' }}>
      <Breadcrumb style={{ margin: '16px 0' }}>
        <Breadcrumb.Item href="#home">Home</Breadcrumb.Item>
        <Breadcrumb.Item>{page}</Breadcrumb.Item>
      </Breadcrumb>
      <div className="site-layout-content">
        <DatabasePage />
      </div>
    </Content>
  )
}

function AppHeader() {
  return (
    <Header>
      <Menu theme="dark" mode="horizontal" defaultSelectedKeys={['db']}>
        <Menu.Item key="logo">
          <img src={logo} className="App-logo" alt="logo" />
          LeifDb
        </Menu.Item>
        <Menu.Item key="db">Database</Menu.Item>
        <Menu.Item key="admin">Admin</Menu.Item>
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

function App() {
  // Get currently selected header tab and send it to AppContent props
  // to choose which page content to display

  // const colors = [0, 0.1, 0.25, 0.5, 0.75, 0.9, 1].map(scp.magma);
  // colors.forEach((i) => {console.log(i)});

  return (
    <Layout className="layout">
      <AppHeader />
      <AppContent />
      <AppFooter />
    </Layout>
  );
}

export default App;
