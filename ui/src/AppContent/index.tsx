import React, { useState, useEffect } from 'react';

import { Layout, Breadcrumb } from 'antd';

import { Page } from '../proptypes';
import DatabasePage from './DatabasePage';
import AdminPage from './AdminPage';
import HomePage from './HomePage';

const { Content } = Layout;

export interface AppContentProps {
  page: Page;
}

export default function AppContent(props:AppContentProps) {
  const [host, setHost] = useState({address: "", healthy: false});
  const [schema, setSchema] = useState({});

  useEffect(() => {
    if (host && host.healthy) {
      const query = `http://${host.address}/`
      fetch(query)
        .then(res => {
          if (!res.ok) {
            throw Error(res.statusText);
          }
          return res;
        })
        .then(res => res.json())
        .then(d => setSchema(d))
        .catch(() => console.log(`Failed to fetch scema from ${host.address}`));
    }
  }, [host])

  useEffect(() => {
    console.log("Schema:", JSON.stringify(schema, null, 2));
  }, [schema])

  const pages: Record<Page, JSX.Element> = {
    Home: <HomePage
      // schema={schema}
    />,
    Database: <DatabasePage
      host={host.address}
      connected={host.healthy}/>,
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
