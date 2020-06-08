import React, { useState } from 'react';

import { Layout, Breadcrumb } from 'antd';

import { AppContentProps, Page } from '../proptypes';
import { DatabasePage } from './DatabasePage';
import { AdminPage } from './AdminPage';
import { HomePage } from './HomePage';

const { Content } = Layout;

export function AppContent(props:AppContentProps) {
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
