import React, { useState } from 'react';
import { Input, Button, Space, Alert, Select } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined, CheckCircleFilled, WarningFilled } from '@ant-design/icons';
import { Server } from '../../proptypes';

const { TextArea } = Input;
const { Option } = Select;

export type Mode = 'ModeSearch' | 'ModeSet' | 'ModeDelete';

export interface DbPageProps {
  host: Server;
}

export default function DatabasePage(props: DbPageProps) {
  const [kv, setKV] = useState({key: "", value: "", error: ""});
  const [mode, setMode] = useState<Mode>("ModeSearch");

  function connectHeader() {
    if (props.host.address) {
      return (
        <span>{props.host.healthy ? <span><CheckCircleFilled /> Connected to</span> : <span><WarningFilled /> Could not connect to</span>} <code>{props.host.address}</code></span>
      )
    }
    return (
      <Alert
        type="error"
        message="Not connected to a database--use the Admin tab to connect"
        showIcon
        closable />
    )
  }

  function errorHeader() {
    if (kv.error) {
      return (
        <Alert
          type="error"
          message={`Error communicating with server: ${kv.error}`}
          showIcon
          closable />
      )
    }
  }

  function searchHandler(key: string) {
    const query = `http://${props.host.address}/db/${key}`
    console.log("GET " + query)
    fetch(query)
      .then(response => {
        return response.text()
      })
      .then(data => {
        setKV({key: kv.key, value: data, error: ""});
      })
      .catch(err => {
        setKV({key: kv.key, value: "", error: err.message});
      })
  }

  function setHandler(key: string) {
    const query = `http://${props.host.address}/db/${key}?value=${encodeURI(kv.value)}`
    console.log("PUT " + query)
    fetch(query, {method: 'PUT'})
      .then(response => response.text())
      .then(data => {
        console.log("Result:", data);
      })
      .catch(err => setKV({key: kv.key, value: kv.value, error: err.message}));
  }

  function deleteHandler(key: string) {
    const query = `http://${props.host.address}/db/${key}`
    console.log("DELETE " + query)
    fetch(query, {method: 'DELETE'})
      .then(res => { console.log("Ok?", res.ok); return res })
      .then(() => setKV({key: kv.key, value: "", error: ""}))
      .catch(err => setKV({key: kv.key, value: "", error: err.message}));
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
    ModeSearch: <TextArea
      className="App-search-field"
      id="result-textarea"
      data-testid="result-textarea"
      value={kv.value} />,
    ModeSet:    <TextArea
      className="App-search-field"
      id="result-textarea"
      data-testid="result-textarea"
      onChange={e => {setKV({key: kv.key, value: e.target.value, error: ""})}}
      allowClear />,
    ModeDelete: <TextArea
      className="App-search-field"
      id="result-textarea"
      data-testid="result-textarea"
      value={kv.value} />
  }

  return (
    <Space direction="vertical">
      {connectHeader()}
      {errorHeader()}
      <Input.Group compact className="App-search-field">
        <Select
          defaultValue="ModeSearch"
          data-testid="mode-dropdown"
          onSelect={val => setMode(val)}
        >
          <Option value="ModeSearch" data-testid="mode-search" ><SearchOutlined />Search</Option>
          <Option value="ModeSet" data-testid="mode-set" ><SaveOutlined /> Set</Option>
          <Option value="ModeDelete" data-testid="mode-delete" ><DeleteOutlined />Delete</Option>
        </Select>
        <Input.Search
          className="App-search-field"
          data-testid="App-key-input"
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
        {buttons.map(d => {
           return <Button
              className="db-button"
              id={d.id}
              key={d.id}
              onClick={d.handler}
            >
              {d.icon} {d.text}
            </Button>})}
      </div>
    </Space>
  )
}
