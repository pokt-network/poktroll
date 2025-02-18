import React from "react";
import SwaggerUI from "swagger-ui-react";
import "swagger-ui-react/swagger-ui.css";

const OpenAPI = () => {
    return (
        <SwaggerUI url="https://raw.githubusercontent.com/pokt-network/poktroll/main/docs/static/openapi.yml" />
    );
};

export default OpenAPI;