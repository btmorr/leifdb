import React from 'react';
import { Input, Space, Card } from 'antd';
import { CheckCircleFilled, WarningFilled } from '@ant-design/icons';
import * as Client from '../../leifDbClientAPI';
import { Server } from '../../proptypes';

const { Search } = Input;

export interface AdminPageProps {
  currentHost: Server;
  setHost: React.Dispatch<React.SetStateAction<Server>>;
}

export default function AdminPage(props:AdminPageProps) {

  function tryConnect(
    address: string,
    handler: React.Dispatch<React.SetStateAction<Server>>
  ) {
    const client = new Client.LeifDbClientAPI({baseUri: `http://${address}`});
    client.httpHealth()
      .then(res => {
        if (res.status !== "Ok") {
          throw Error(`Health status ${res.status}`);
        }
        return res;
      })
      .then(res => handler({
        address: address,
        healthy: res.status === "Ok",
        client: client
      }))
      .catch(() => handler({
        address: address,
        healthy: false}));
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
        className="App-search-field"
        placeholder="localhost:8080"
        onSearch={value => { tryConnect(value, props.setHost) }}
      />
      <span>
        Address should be in the form "<code>host:port</code>",
        without quote marks (example: <code>192.168.0.3:8080</code>)
      </span>
      {buildCard()}
    </Space>
  )
}
