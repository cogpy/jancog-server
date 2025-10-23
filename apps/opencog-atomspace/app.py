"""
OpenCog AtomSpace Microservice

This service provides a REST API for the OpenCog AtomSpace hypergraph database.
AtomSpace stores atoms (terms, formulas, sentences) and their relationships.
"""

from flask import Flask, request, jsonify
from flask_cors import CORS
import logging
import os
from datetime import datetime
from typing import Dict, List, Any
import json

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)

app = Flask(__name__)
CORS(app)

# In-memory AtomSpace storage (simplified implementation)
# In production, this would use the actual OpenCog AtomSpace C++ library
atomspace = {
    'atoms': {},
    'relationships': [],
    'atom_counter': 0
}


class Atom:
    """Represents an atom in the AtomSpace"""
    
    def __init__(self, atom_type: str, name: str, truth_value: Dict[str, float] = None):
        self.id = atomspace['atom_counter']
        atomspace['atom_counter'] += 1
        self.type = atom_type
        self.name = name
        self.truth_value = truth_value or {'strength': 1.0, 'confidence': 1.0}
        self.created_at = datetime.utcnow().isoformat()
        
    def to_dict(self) -> Dict[str, Any]:
        return {
            'id': self.id,
            'type': self.type,
            'name': self.name,
            'truth_value': self.truth_value,
            'created_at': self.created_at
        }


@app.route('/healthcheck', methods=['GET'])
def healthcheck():
    """Health check endpoint"""
    return jsonify({
        'status': 'healthy',
        'service': 'opencog-atomspace',
        'timestamp': datetime.utcnow().isoformat(),
        'atom_count': len(atomspace['atoms']),
        'relationship_count': len(atomspace['relationships'])
    }), 200


@app.route('/v1/version', methods=['GET'])
def version():
    """Version information endpoint"""
    return jsonify({
        'service': 'opencog-atomspace',
        'version': '0.1.0',
        'opencog_version': 'compatible',
        'description': 'OpenCog AtomSpace hypergraph database service'
    }), 200


@app.route('/api/v1/atoms', methods=['POST'])
def create_atom():
    """
    Create a new atom in the AtomSpace
    
    Request body:
    {
        "type": "ConceptNode",
        "name": "human",
        "truth_value": {"strength": 0.9, "confidence": 0.8}
    }
    """
    try:
        data = request.get_json()
        
        if not data or 'type' not in data or 'name' not in data:
            return jsonify({
                'error': 'Missing required fields: type and name'
            }), 400
        
        atom = Atom(
            atom_type=data['type'],
            name=data['name'],
            truth_value=data.get('truth_value')
        )
        
        atomspace['atoms'][atom.id] = atom
        
        logger.info(f"Created atom: {atom.type} - {atom.name} (ID: {atom.id})")
        
        return jsonify({
            'success': True,
            'atom': atom.to_dict()
        }), 201
        
    except Exception as e:
        logger.error(f"Error creating atom: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/atoms/<int:atom_id>', methods=['GET'])
def get_atom(atom_id: int):
    """Get an atom by ID"""
    try:
        if atom_id not in atomspace['atoms']:
            return jsonify({'error': 'Atom not found'}), 404
        
        atom = atomspace['atoms'][atom_id]
        return jsonify({
            'success': True,
            'atom': atom.to_dict()
        }), 200
        
    except Exception as e:
        logger.error(f"Error getting atom: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/atoms', methods=['GET'])
def list_atoms():
    """List all atoms in the AtomSpace"""
    try:
        atom_type = request.args.get('type')
        
        atoms = []
        for atom in atomspace['atoms'].values():
            if atom_type is None or atom.type == atom_type:
                atoms.append(atom.to_dict())
        
        return jsonify({
            'success': True,
            'count': len(atoms),
            'atoms': atoms
        }), 200
        
    except Exception as e:
        logger.error(f"Error listing atoms: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/atoms/<int:atom_id>', methods=['DELETE'])
def delete_atom(atom_id: int):
    """Delete an atom from the AtomSpace"""
    try:
        if atom_id not in atomspace['atoms']:
            return jsonify({'error': 'Atom not found'}), 404
        
        atom = atomspace['atoms'][atom_id]
        del atomspace['atoms'][atom_id]
        
        logger.info(f"Deleted atom: {atom.type} - {atom.name} (ID: {atom_id})")
        
        return jsonify({
            'success': True,
            'message': f'Atom {atom_id} deleted successfully'
        }), 200
        
    except Exception as e:
        logger.error(f"Error deleting atom: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/relationships', methods=['POST'])
def create_relationship():
    """
    Create a relationship between atoms
    
    Request body:
    {
        "type": "InheritanceLink",
        "outgoing": [atom_id1, atom_id2],
        "truth_value": {"strength": 0.9, "confidence": 0.8}
    }
    """
    try:
        data = request.get_json()
        
        if not data or 'type' not in data or 'outgoing' not in data:
            return jsonify({
                'error': 'Missing required fields: type and outgoing'
            }), 400
        
        # Verify all atoms exist
        for atom_id in data['outgoing']:
            if atom_id not in atomspace['atoms']:
                return jsonify({
                    'error': f'Atom {atom_id} not found'
                }), 404
        
        relationship = {
            'id': len(atomspace['relationships']),
            'type': data['type'],
            'outgoing': data['outgoing'],
            'truth_value': data.get('truth_value', {'strength': 1.0, 'confidence': 1.0}),
            'created_at': datetime.utcnow().isoformat()
        }
        
        atomspace['relationships'].append(relationship)
        
        logger.info(f"Created relationship: {relationship['type']} between atoms {relationship['outgoing']}")
        
        return jsonify({
            'success': True,
            'relationship': relationship
        }), 201
        
    except Exception as e:
        logger.error(f"Error creating relationship: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/relationships', methods=['GET'])
def list_relationships():
    """List all relationships in the AtomSpace"""
    try:
        return jsonify({
            'success': True,
            'count': len(atomspace['relationships']),
            'relationships': atomspace['relationships']
        }), 200
        
    except Exception as e:
        logger.error(f"Error listing relationships: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/query', methods=['POST'])
def query_atomspace():
    """
    Query the AtomSpace for patterns
    
    Request body:
    {
        "pattern": {
            "type": "ConceptNode",
            "name": "human"
        }
    }
    """
    try:
        data = request.get_json()
        
        if not data or 'pattern' not in data:
            return jsonify({
                'error': 'Missing required field: pattern'
            }), 400
        
        pattern = data['pattern']
        results = []
        
        for atom in atomspace['atoms'].values():
            match = True
            if 'type' in pattern and atom.type != pattern['type']:
                match = False
            if 'name' in pattern and atom.name != pattern['name']:
                match = False
            
            if match:
                results.append(atom.to_dict())
        
        return jsonify({
            'success': True,
            'count': len(results),
            'results': results
        }), 200
        
    except Exception as e:
        logger.error(f"Error querying AtomSpace: {str(e)}")
        return jsonify({'error': str(e)}), 500


@app.route('/api/v1/clear', methods=['POST'])
def clear_atomspace():
    """Clear all atoms and relationships from the AtomSpace"""
    try:
        atom_count = len(atomspace['atoms'])
        relationship_count = len(atomspace['relationships'])
        
        atomspace['atoms'].clear()
        atomspace['relationships'].clear()
        atomspace['atom_counter'] = 0
        
        logger.info(f"Cleared AtomSpace: {atom_count} atoms, {relationship_count} relationships")
        
        return jsonify({
            'success': True,
            'message': f'Cleared {atom_count} atoms and {relationship_count} relationships'
        }), 200
        
    except Exception as e:
        logger.error(f"Error clearing AtomSpace: {str(e)}")
        return jsonify({'error': str(e)}), 500


if __name__ == '__main__':
    port = int(os.environ.get('PORT', 8100))
    host = os.environ.get('HOST', '0.0.0.0')
    
    logger.info(f"Starting OpenCog AtomSpace service on {host}:{port}")
    app.run(host=host, port=port, debug=False)
