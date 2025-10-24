"""
OpenCog CogServer Microservice

This service provides a container and scheduler for plug-in cognitive algorithms.
It manages the execution and coordination of various AI processes.
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
import logging
import os
from datetime import datetime
from typing import Dict, List, Any
import json
import threading
import time
from enum import Enum

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)


class AgentStatus(Enum):
    """Status of cognitive agents"""
    STOPPED = "stopped"
    RUNNING = "running"
    PAUSED = "paused"
    ERROR = "error"


class CognitiveAgent:
    """Represents a cognitive agent in the CogServer"""
    
    def __init__(self, name: str, agent_type: str, config: Dict[str, Any] = None):
        self.name = name
        self.type = agent_type
        self.config = config or {}
        self.status = AgentStatus.STOPPED
        self.created_at = datetime.utcnow().isoformat()
        self.last_run = None
        self.run_count = 0
        self.errors = []
        
    def to_dict(self) -> Dict[str, Any]:
        return {
            'name': self.name,
            'type': self.type,
            'config': self.config,
            'status': self.status.value,
            'created_at': self.created_at,
            'last_run': self.last_run,
            'run_count': self.run_count,
            'error_count': len(self.errors)
        }


# CogServer state
cogserver = {
    'agents': {},
    'scheduler_running': False,
    'scheduler_thread': None
}


def scheduler_loop():
    """Main scheduler loop for running cognitive agents"""
    logger.info("Scheduler thread started")
    
    while cogserver['scheduler_running']:
        try:
            # Run each active agent
            for name, agent in cogserver['agents'].items():
                if agent.status == AgentStatus.RUNNING:
                    try:
                        # Simulate agent execution
                        agent.last_run = datetime.utcnow().isoformat()
                        agent.run_count += 1
                        logger.debug(f"Agent {name} executed (run #{agent.run_count})")
                    except Exception as e:
                        logger.error(f"Error running agent {name}: {str(e)}")
                        agent.errors.append({
                            'timestamp': datetime.utcnow().isoformat(),
                            'error': str(e)
                        })
                        agent.status = AgentStatus.ERROR
            
            # Sleep between cycles
            time.sleep(1)
            
        except Exception as e:
            logger.error(f"Scheduler error: {str(e)}")
    
    logger.info("Scheduler thread stopped")


@app.route('/healthcheck', methods=['GET'])
def healthcheck():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'opencog-cogserver',
        'timestamp': datetime.utcnow().isoformat(),
        'agent_count': len(cogserver['agents']),
        'scheduler_running': cogserver['scheduler_running']
    }), 200


@app.route('/v1/version', methods=['GET'])
def version():
    """Version information endpoint"""
    return jsonify({
        'service': 'opencog-cogserver',
        'version': '0.1.0',
        'opencog_version': 'compatible',
        'description': 'OpenCog CogServer cognitive algorithm scheduler'
    }), 200


@app.route('/api/v1/agents', methods=['POST'])
def create_agent():
    """
    Create a new cognitive agent
    
    Request body:
    {
        "name": "pattern_matcher",
        "type": "PatternMatchingAgent",
        "config": {"threshold": 0.5}
    }
    """
    try:
        data = request.get_json()
        
        if not data or 'name' not in data or 'type' not in data:
            return jsonify({
                'error': 'Missing required fields: name and type'
            }), 400
        
        if data['name'] in cogserver['agents']:
            return jsonify({
                'error': f'Agent {data["name"]} already exists'
            }), 409
        
        agent = CognitiveAgent(
            name=data['name'],
            agent_type=data['type'],
            config=data.get('config')
        )
        
        cogserver['agents'][agent.name] = agent
        
        logger.info(f"Created agent: {agent.name} ({agent.type})")
        
        return jsonify({
            'success': True,
            'agent': agent.to_dict()
        }), 201
        
    except Exception as e:
        logger.error(f"Error creating agent: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/agents/<string:name>', methods=['GET'])
def get_agent(name: str):
    """Get an agent by name"""
    try:
        if name not in cogserver['agents']:
            return jsonify({'error': 'Agent not found'}), 404
        
        agent = cogserver['agents'][name]
        return jsonify({
            'success': True,
            'agent': agent.to_dict()
        }), 200
        
    except Exception as e:
        logger.error(f"Error getting agent: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/agents', methods=['GET'])
def list_agents():
    """List all cognitive agents"""
    try:
        agents = [agent.to_dict() for agent in cogserver['agents'].values()]
        
        return jsonify({
            'success': True,
            'count': len(agents),
            'agents': agents
        }), 200
        
    except Exception as e:
        logger.error(f"Error listing agents: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/agents/<string:name>', methods=['DELETE'])
def delete_agent(name: str):
    """Delete a cognitive agent"""
    try:
        if name not in cogserver['agents']:
            return jsonify({'error': 'Agent not found'}), 404
        
        agent = cogserver['agents'][name]
        
        # Stop agent if running
        if agent.status == AgentStatus.RUNNING:
            agent.status = AgentStatus.STOPPED
        
        del cogserver['agents'][name]
        
        logger.info(f"Deleted agent: {name}")
        
        return jsonify({
            'success': True,
            'message': f'Agent {name} deleted successfully'
        }), 200
        
    except Exception as e:
        logger.error(f"Error deleting agent: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/agents/<string:name>/start', methods=['POST'])
def start_agent(name: str):
    """Start a cognitive agent"""
    try:
        if name not in cogserver['agents']:
            return jsonify({'error': 'Agent not found'}), 404
        
        agent = cogserver['agents'][name]
        
        if agent.status == AgentStatus.RUNNING:
            return jsonify({
                'success': True,
                'message': 'Agent is already running'
            }), 200
        
        agent.status = AgentStatus.RUNNING
        
        # Start scheduler if not running
        if not cogserver['scheduler_running']:
            cogserver['scheduler_running'] = True
            cogserver['scheduler_thread'] = threading.Thread(target=scheduler_loop, daemon=True)
            cogserver['scheduler_thread'].start()
        
        logger.info(f"Started agent: {name}")
        
        return jsonify({
            'success': True,
            'agent': agent.to_dict()
        }), 200
        
    except Exception as e:
        logger.error(f"Error starting agent: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/agents/<string:name>/stop', methods=['POST'])
def stop_agent(name: str):
    """Stop a cognitive agent"""
    try:
        if name not in cogserver['agents']:
            return jsonify({'error': 'Agent not found'}), 404
        
        agent = cogserver['agents'][name]
        agent.status = AgentStatus.STOPPED
        
        logger.info(f"Stopped agent: {name}")
        
        return jsonify({
            'success': True,
            'agent': agent.to_dict()
        }), 200
        
    except Exception as e:
        logger.error(f"Error stopping agent: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/scheduler/start', methods=['POST'])
def start_scheduler():
    """Start the cognitive agent scheduler"""
    try:
        if cogserver['scheduler_running']:
            return jsonify({
                'success': True,
                'message': 'Scheduler is already running'
            }), 200
        
        cogserver['scheduler_running'] = True
        cogserver['scheduler_thread'] = threading.Thread(target=scheduler_loop, daemon=True)
        cogserver['scheduler_thread'].start()
        
        logger.info("Scheduler started")
        
        return jsonify({
            'success': True,
            'message': 'Scheduler started successfully'
        }), 200
        
    except Exception as e:
        logger.error(f"Error starting scheduler: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/scheduler/stop', methods=['POST'])
def stop_scheduler():
    """Stop the cognitive agent scheduler"""
    try:
        if not cogserver['scheduler_running']:
            return jsonify({
                'success': True,
                'message': 'Scheduler is not running'
            }), 200
        
        cogserver['scheduler_running'] = False
        
        # Stop all agents
        for agent in cogserver['agents'].values():
            if agent.status == AgentStatus.RUNNING:
                agent.status = AgentStatus.STOPPED
        
        logger.info("Scheduler stopped")
        
        return jsonify({
            'success': True,
            'message': 'Scheduler stopped successfully'
        }), 200
        
    except Exception as e:
        logger.error(f"Error stopping scheduler: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/scheduler/status', methods=['GET'])
def scheduler_status():
    """Get scheduler status"""
    try:
        running_agents = sum(1 for agent in cogserver['agents'].values() 
                            if agent.status == AgentStatus.RUNNING)
        
        return jsonify({
            'success': True,
            'scheduler': {
                'running': cogserver['scheduler_running'],
                'total_agents': len(cogserver['agents']),
                'running_agents': running_agents
            }
        }), 200
        
    except Exception as e:
        logger.error(f"Error getting scheduler status: {str(e)}")
        return jsonify({'error': str(e)}), 500


if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8101))
    host = os.environ.get('HOST', '0.0.0.0')
    
    logger.info(f"Starting OpenCog CogServer service on {host}:{port}")
    app.run(host=host, port=port, debug=False)
