pipeline {
  agent any

  environment {
    dockerImage = ''
    imageName = 'judgegodwins/chess-server'
    registryCred = 'dockerhub-cred'
  }

  stages {
    stage('build') {
      steps {
        echo 'Building app'
        script {
            image = "${imageName}:${env.BUILD_ID}"
            dockerImage = docker.build(image)
        }
      }
    }

    stage('test') {
      steps {
        echo 'Testing app'
      }
    }

    stage('deploy') {
      steps {
        echo 'Deploying app'
        script {
          docker.withRegistry('', registryCred) {
            dockerImage.push()
            dockerImage.push('latest')
          }
        }
      }
    }
  }
}
