import React from 'react';
import { Input, Space, Card } from 'antd';
import { CheckCircleFilled, WarningFilled } from '@ant-design/icons';
import { AdminPageProps, Server } from '../../proptypes';

const { Search } = Input;

export default function AdminPage(props:AdminPageProps) {

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
