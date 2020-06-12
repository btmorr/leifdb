import React from 'react';
import logo from '../logo.svg';
import { Layout, Menu } from 'antd';

import { Page } from '../proptypes';

const { Header } = Layout;

export interface HeaderProps {
  page: Page;
  clickHandler: (p: Page) => void;
}

export default function AppHeader(props:HeaderProps) {

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
