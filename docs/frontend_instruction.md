##Swagger file

openapi: 3.0.3

info:
  title: Risk Detection API
  description: Frontend API documentation for the Risk Detection System
  version: 1.0.0

servers:
  - url: http://localhost:8080
    description: Local development server

tags:
  - name: Authentication
    description: Signup and Login APIs
  - name: Transaction
    description: Transaction and risk evaluation APIs

paths:
  /v1/signup:
    post:
      tags:
        - Authentication
      summary: Signup new user
      description: Register a new user and return a JWT token
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/SignupRequest"
            example:
              email: user@example.com
              password: SecurePassword123
              role: USER
              device_id: device-123-abc
      responses:
        "200":
          description: Signup successful
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "400":
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"

  /v1/login:
    post:
      tags:
        - Authentication
      summary: Login user
      description: Authenticate user and return JWT token
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/LoginRequest"
            example:
              email: user@example.com
              password: SecurePassword123
              device_id: device-123-abc
      responses:
        "200":
          description: Login successful
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/AuthResponse"
        "401":
          description: Invalid credentials
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"

  /api/v1/transaction:
    post:
      tags:
        - Transaction
      summary: Create transaction and evaluate risk
      description: >
        Creates a transaction and evaluates its risk.
        Requires JWT authentication.
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: "#/components/schemas/TransactionRequest"
            example:
              transaction_type: TRANSFER
              receiver_id: 3fa85f64-5717-4562-b3fc-2c963f66afa6
              amount: 1500.75
              device_id: device123
              transaction_time: 2023-04-10T12:34:56Z
      responses:
        "200":
          description: Transaction risk evaluated
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/TransactionRiskResponse"
        "400":
          description: Invalid request payload
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "401":
          description: Unauthorized
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "403":
          description: Transaction blocked due to high risk
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    SignupRequest:
      type: object
      required:
        - email
        - password
        - role
        - device_id
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          format: password
        role:
          type: string
          example: USER
        device_id:
          type: string

    LoginRequest:
      type: object
      required:
        - email
        - password
        - device_id
      properties:
        email:
          type: string
          format: email
        password:
          type: string
          format: password
        device_id:
          type: string

    AuthResponse:
      type: object
      properties:
        access_token:
          type: string
        token_type:
          type: string
          example: Bearer
        expires_in:
          type: integer
          example: 3600

    TransactionRequest:
      type: object
      required:
        - transaction_type
        - amount
        - device_id
        - transaction_time
      properties:
        transaction_type:
          type: string
          example: TRANSFER
        receiver_id:
          type: string
          format: uuid
          nullable: true
        amount:
          type: number
          format: double
          minimum: 0.01
        device_id:
          type: string
        transaction_time:
          type: string
          format: date-time

    TransactionRiskResponse:
      type: object
      properties:
        risk_result:
          type: object
          properties:
            transaction_id:
              type: string
              format: uuid
            risk_score:
              type: integer
            risk_level:
              type: string
              example: MEDIUM
            decision:
              type: string
              example: FLAG
            evaluated_at:
              type: string
              format: date-time

    ErrorResponse:
      type: object
      properties:
        error:
          type: string
