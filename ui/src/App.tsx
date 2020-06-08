import React, { FunctionComponent, useState } from 'react';
import logo from './logo.svg';
import { Layout, Menu, Breadcrumb, Input, Button, Space, Alert, Card, Select } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined, CheckCircleFilled, WarningFilled } from '@ant-design/icons';
// import * as scp from 'scale-color-perceptual';
import './App.css';

const { Header, Content, Footer } = Layout;
const { TextArea, Search } = Input;
const { Option } = Select;

type Mode = 'ModeSearch' | 'ModeSet' | 'ModeDelete';

interface DbPageProps {
  host: Server;
}

function DatabasePage(props: DbPageProps) {
  const [kv, setKV] = useState({key: "", value: ""});
  const [mode, setMode] = useState<Mode>("ModeSearch");

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

  function searchHandler(key: string) {
    const query = `http://${props.host.address}/db/${key}`
    console.log("GET " + query)
    fetch(query)
      .then(response => response.text())
      .then(data => {
        console.log("Result:", data);
        setKV({key: kv.key, value: data});
      });
  }

  function setHandler(key: string) {
    const query = `http://${props.host.address}/db/${key}?value=${encodeURI(kv.value)}`
    console.log("PUT " + query)
    fetch(query, {method: 'PUT'})
      .then(response => response.text())
      .then(data => {
        console.log("Result:", data);
      });
  }

  function deleteHandler(key: string) {
    const query = `http://${props.host.address}/db/${key}`
    console.log("DELETE " + query)
    // todo: improve error handling. if !res.ok show an alert and don't clear results
    fetch(query, {method: 'DELETE'})
      .then(res => { console.log("Ok?", res.ok); return res })
      .then(() => setKV({key: kv.key, value: ""}))
  }

  const buttons = [
    {
      id: "copy-button",
      text: "Copy to clipboard",
      icon: <CopyOutlined />,
      handler: () => {
        const textarea: HTMLInputElement = document.getElementById("result-textarea")! as HTMLInputElement;
        textarea.select();
        document.execCommand('copy');
      }
    },
    // {
    //   id: "save-button",
    //   text: "Save to database",
    //   icon: <SaveOutlined />,
    //   handler: () => {console.log("save button")}
    // },
    // {
    //   id: "delete-button",
    //   text: "Delete from database",
    //   icon: <DeleteOutlined />,
    //   handler: () => {
    //     console.log("delete button");
    //     deleteHandler(kv.key);
    //   }
    // }
  ];

  const resultElement: Record<Mode, JSX.Element> = {
    ModeSearch: <TextArea className="App-result-field" id="result-textarea" value={kv.value} />,
    ModeSet:    <TextArea className="App-result-field" id="result-textarea" onChange={e => {setKV({key: kv.key, value: e.target.value})}} allowClear />,
    ModeDelete: <TextArea className="App-result-field" id="result-textarea" value={kv.value} />
  }

  return (
    <Space direction="vertical">
      {connectHeader()}
      <Input.Group compact>
        <Select
          defaultValue="ModeSearch"
          onSelect={(val, opt) => setMode(val)}
        >
          <Option value="ModeSearch"><SearchOutlined />Search</Option>
          <Option value="ModeSet"><SaveOutlined /> Set</Option>
          <Option value="ModeDelete"><DeleteOutlined />Delete</Option>
        </Select>
        <Input.Search
          className="App-key-input"
          placeholder="Search for a key here..."
          onSearch={key => {
            switch(mode) {
              case "ModeSearch": searchHandler(key); break;
              case "ModeSet": setHandler(key); break;
              case "ModeDelete": deleteHandler(key); break;
              default: console.log("invalid select option:", mode);
            }
          }}
          allowClear
        />
      </Input.Group>
      {resultElement[mode]}
      <div>
        {buttons.map(d => { return <Button className="db-button" id={d.id} onClick={d.handler}>{d.icon} {d.text}</Button>})}
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
