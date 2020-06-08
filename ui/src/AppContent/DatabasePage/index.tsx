import React, { useState } from 'react';
import { Input, Button, Space, Alert, Select } from 'antd';
import { SearchOutlined, CopyOutlined, SaveOutlined, DeleteOutlined, CheckCircleFilled, WarningFilled } from '@ant-design/icons';
import { DbPageProps, Mode } from '../../proptypes';

const { TextArea } = Input;
const { Option } = Select;

export function DatabasePage(props: DbPageProps) {
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
