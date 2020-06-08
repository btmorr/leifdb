import React, { FunctionComponent, useState } from 'react';
import { Layout } from 'antd';
// import * as scp from 'scale-color-perceptual';

import { AppHeader } from './AppHeader';
import { AppContent } from './AppContent';
import { AppFooter } from './AppFooter';

import { Page } from './proptypes';

import './App.css';

const App:FunctionComponent<{ initialPage?: Page }> = ({ initialPage = "Admin" }) => {
  const [currentPage, setCurrentPage] = useState(initialPage);
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
